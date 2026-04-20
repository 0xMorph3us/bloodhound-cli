package internal

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// Vars for tracking the list of BloodHound images
// Used for filtering the list of containers returned by the Docker client
var (
	prodImages = []string{
		"bhce_bloodhound", "bhce_neo4j", "bhce_postgres",
	}
	devImages = []string{
		"bhce_bloodhound", "bhce_neo4j", "bhce_postgres",
	}
	// Default root command for Docker commands, will fallback to Podman if Docker is not found
	dockerCmd = "docker"
	// URLs for the BloodHound compose files
	devYaml  = "docker-compose.dev.yml"
	prodYaml = "docker-compose.yml"
	loginUri = "/ui/login"
)

func getTemplateDir() string {
	return filepath.Join(GetBloodHoundDir(), "template")
}

func copyFile(src string, dst string) {
	content, readErr := os.ReadFile(src)
	if readErr != nil {
		log.Fatalf("Error trying to read %s: %v\n", src, readErr)
	}
	writeErr := os.WriteFile(dst, content, 0644)
	if writeErr != nil {
		log.Fatalf("Error trying to write %s: %v\n", dst, writeErr)
	}
}

// InstallGDSPluginFile copies a local GDS plugin jar into the config directory's
// plugins path so Docker Compose can mount it at /plugins/graph-data-science.jar.
func InstallGDSPluginFile(src string) string {
	if !FileExists(src) {
		log.Fatalf("GDS plugin file does not exist: %s", src)
	}

	pluginsDir := filepath.Join(GetBloodHoundDir(), "plugins")
	mkErr := os.MkdirAll(pluginsDir, 0755)
	if mkErr != nil {
		log.Fatalf("Error creating plugins directory %s: %v", pluginsDir, mkErr)
	}

	dst := filepath.Join(pluginsDir, "graph-data-science.jar")
	copyFile(src, dst)
	fmt.Printf("[+] Installed GDS plugin jar to %s\n", dst)
	return dst
}

// ensureTemplateFile returns a template file path under the configured template directory.
// Priority order:
// 1. Template next to the bloodhound-cli binary (when refreshFromExecutable is true)
// 2. Existing template in config_directory/template (user-managed)
// 3. Template next to the bloodhound-cli binary
func ensureTemplateFile(filename string, refreshFromExecutable bool) string {
	templateDir := getTemplateDir()
	mkErr := os.MkdirAll(templateDir, 0755)
	if mkErr != nil {
		log.Fatalf("Error trying to create template directory %s: %v\n", templateDir, mkErr)
	}

	templatePath := filepath.Join(templateDir, filename)
	exeTemplatePath := filepath.Join(GetCwdFromExe(), filename)
	if refreshFromExecutable && FileExists(exeTemplatePath) {
		copyFile(exeTemplatePath, templatePath)
		fmt.Printf("[+] Refreshed YAML template from %s\n", exeTemplatePath)
		return templatePath
	}

	if FileExists(templatePath) {
		return templatePath
	}

	if FileExists(exeTemplatePath) {
		copyFile(exeTemplatePath, templatePath)
		fmt.Printf("[+] Installed local YAML template to %s\n", templatePath)
		return templatePath
	}

	log.Fatalf(
		"Missing template file %s. Please place it in %s or next to the bloodhound-cli binary.",
		filename,
		templateDir,
	)
	return ""
}

func syncComposeFromTemplate(filename string, force bool, refreshTemplate bool, promptLabel string) {
	templatePath := ensureTemplateFile(filename, refreshTemplate)
	dst := filepath.Join(GetBloodHoundDir(), filename)

	shouldWrite := force || !FileExists(dst)
	if !shouldWrite {
		c := AskForConfirmation(promptLabel)
		shouldWrite = c
	}

	if shouldWrite {
		copyFile(templatePath, dst)
		fmt.Printf("[+] Synced %s from template %s\n", dst, templatePath)
	}
}

// Container is a custom type for storing container information similar to output from "docker containers ls".
type Container struct {
	ID     string
	Image  string
	Status string
	Ports  []container.PortSummary
	Name   string
}

// Containers is a collection of Container structs
type Containers []Container

// Len returns the length of a Containers struct
func (c Containers) Len() int {
	return len(c)
}

// Less determines if one Container is less than another Container
func (c Containers) Less(i, j int) bool {
	return c[i].Image < c[j].Image
}

// Swap exchanges the position of two Container values in a Containers struct
func (c Containers) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

// EvaluateDockerComposeStatus checks if Docker (or Podman in Docker compatibility mode) and the Docker Compose plugin are installed and operational.
// It verifies the presence of the CLI, ensures the daemon is running, and sets the global dockerCmd variable to either `docker` or `podman`.
// The function exits fatally via log.Fatal* if any requirement is not met; otherwise it returns normally.
func EvaluateDockerComposeStatus() {
	fmt.Println("[+] Checking the status of Docker and the Compose plugin...")
	// Check for ``docker`` first because it's required for everything to come
	dockerExists := CheckPath("docker")
	if !dockerExists {
		podmanExists := CheckPath("podman")
		if podmanExists {
			fmt.Println("[+] Docker is not installed, but Podman is installed. Using Podman as a Docker alternative.")
			dockerCmd = "podman"
		} else {
			log.Fatalln("Neither Docker nor Podman is installed on this system, so please install Docker or Podman (in Docker compatibility mode) and try again.")
		}
	}

	// Check if the Docker Engine is running
	_, engineErr := RunBasicCmd(dockerCmd, []string{"info"})
	if engineErr != nil {
		log.Fatalf("%s is installed on this system, but the daemon is not running or access was denied.", dockerCmd)
	}

	// Check for the ``compose`` plugin as our first choice
	_, composeErr := RunBasicCmd(dockerCmd, []string{"compose", "version"})
	if composeErr != nil {
		// Check if the deprecated v1 script is installed
		composeScriptExists := CheckPath("docker-compose")
		if composeScriptExists {
			fmt.Println("[!] The deprecated `docker-compose` v1 script was detected on your system")
			fmt.Println("[!] Docker has deprecated v1 and this CLI tool no longer supports it")
			log.Fatalln("Please upgrade to Docker Compose v2 and try again: https://docs.docker.com/compose/install/")
		} else {
			log.Fatalln("Docker Compose is not installed, so please install it and try again: https://docs.docker.com/compose/install/")
		}
	}

	fmt.Println("[+] Docker and the Compose plugin checks have passed")
}

// DownloadDockerComposeFiles downloads the production and development Docker Compose YAML files into the BloodHound directory.
// If either file already exists, prompts the user for confirmation before overwriting. Exits fatally on download failure.
func DownloadDockerComposeFiles(force bool, refreshTemplate bool) {
	syncComposeFromTemplate(
		prodYaml,
		force,
		refreshTemplate,
		"[*] A production YAML file already exists in the current directory. Do you want to overwrite it?",
	)
	syncComposeFromTemplate(
		devYaml,
		force,
		refreshTemplate,
		"[*] A development YAML file already exists in the current directory. Do you want to overwrite it?",
	)
}

// EvaluateEnvironment checks for the presence of Docker YAML files and initiates their download if necessary.
func EvaluateEnvironment() {
	fmt.Println("[+] Checking for the Docker YAML files...")
	DownloadDockerComposeFiles(false, false)
}

// RunDockerComposeInstall performs a first-time installation of BloodHound containers using the specified Docker Compose YAML file.
// It ensures required YAML files are present, pulls container images, and starts the environment in detached mode.
// Prints login credentials and UI access information upon successful setup. Exits fatally on errors.
func RunDockerComposeInstall(yaml string) {
	// Always sync active compose files from template during install so template edits are applied.
	DownloadDockerComposeFiles(true, true)

	CheckYamlExists(yaml)
	buildErr := RunCmd(dockerCmd, []string{"-f", yaml, "pull"})
	if buildErr != nil {
		log.Fatalf("Error trying to build with %s: %v\n", yaml, buildErr)
	}
	upErr := RunCmd(dockerCmd, []string{"-f", yaml, "up", "-d"})
	if upErr != nil {
		log.Fatalf("Error trying to bring up environment with %s: %v\n", yaml, upErr)
	}
	fmt.Println("[+] BloodHound is ready to go!")
	fmt.Printf("[+] You can log in as `%s` with this password: %s\n", bhEnv.GetString("default_admin.principal_name"), bhEnv.GetString("default_admin.password"))
	fmt.Println("[+] You can get your admin password by running: bloodhound-cli config get default_password")
	fmt.Printf("[+] You can access the BloodHound UI at: %s%s\n", bhEnv.GetString("root_url"), loginUri)
}

// RunDockerComposeUninstall removes all BloodHound containers, images, and volumes defined in the specified Docker
// Compose YAML file, then optionally deletes the BloodHound config directory after user confirmation. The process is
// interactive and exits if the user declines any confirmation prompt. Fatal errors are logged if uninstallation or
// directory deletion fails.
func RunDockerComposeUninstall(yaml string) {
	c := AskForConfirmation("[!] This command removes all containers, images, and volume data. Are you sure you want to uninstall?")
	if !c {
		os.Exit(0)
	}

	fmt.Println("[+] Uninstalling the BloodHound containers...")
	CheckYamlExists(yaml)
	uninstallErr := RunCmd(dockerCmd, []string{"-f", yaml, "down", "--rmi", "all", "-v", "--remove-orphans"})
	if uninstallErr != nil {
		log.Fatalf("Error trying to uninstall with %s: %v\n", yaml, uninstallErr)
	}

	configDir := GetBloodHoundDir()
	delConf := AskForConfirmation("[!] Do you want to also delete the config directory, " + configDir + ", and its contents?")
	if !delConf {
		os.Exit(0)
	}

	delErr := os.RemoveAll(configDir)
	if delErr != nil {
		log.Fatalf("Error trying to delete the config directory: %v\n", delErr)
	} else {
		fmt.Println("[+] Successfully deleted the BloodHound config directory!")
	}
	fmt.Println("[+] Uninstall was successful. You can re-install with `./bloodhound-cli install`.")
	fmt.Println("[+] The config directory and JSON config file will be recreated if you continue using BloodHound CLI.")
}

// RunDockerComposeUpgrade rebuilds and restarts all containers defined in the specified Docker Compose YAML file.
// It brings down any running containers, rebuilds images, and brings the environment back up in detached mode.
// Exits fatally if any Docker command fails.
func RunDockerComposeUpgrade(yaml string) {
	fmt.Printf("[+] Running `%s` commands to build containers with %s...\n", dockerCmd, yaml)
	CheckYamlExists(yaml)
	downErr := RunCmd(dockerCmd, []string{"-f", yaml, "down"})
	if downErr != nil {
		log.Fatalf("Error trying to bring down any running containers with %s: %v\n", yaml, downErr)
	}
	buildErr := RunCmd(dockerCmd, []string{"-f", yaml, "build"})
	if buildErr != nil {
		log.Fatalf("Error trying to build with %s: %v\n", yaml, buildErr)
	}
	upErr := RunCmd(dockerCmd, []string{"-f", yaml, "up", "-d"})
	if upErr != nil {
		log.Fatalf("Error trying to bring up environment with %s: %v\n", yaml, upErr)
	}
	fmt.Println("[+] All containers have been built!")
}

// RunDockerComposeStart starts all services defined in the specified Docker Compose YAML file.
// Exits fatally if the YAML file does not exist or if starting the containers fails.
func RunDockerComposeStart(yaml string) {
	fmt.Printf("[+] Running `%s` to restart containers with %s...\n", dockerCmd, yaml)
	CheckYamlExists(yaml)
	startErr := RunCmd(dockerCmd, []string{"-f", yaml, "start"})
	if startErr != nil {
		log.Fatalf("Error trying to restart the containers with %s: %v\n", yaml, startErr)
	}
}

// RunDockerComposeStop stops all services defined in the specified Docker Compose YAML file.
// Exits the program if stopping services fails.
func RunDockerComposeStop(yaml string) {
	fmt.Printf("[+] Running `%s` to stop services with %s...\n", dockerCmd, yaml)
	CheckYamlExists(yaml)
	stopErr := RunCmd(dockerCmd, []string{"-f", yaml, "stop"})
	if stopErr != nil {
		log.Fatalf("Error trying to stop services with %s: %v\n", yaml, stopErr)
	}
}

// RunDockerComposeRestart restarts all containers defined in the specified Docker Compose YAML file.
// Exits fatally if the YAML file does not exist or if the restart operation fails.
func RunDockerComposeRestart(yaml string) {
	fmt.Printf("[+] Running `%s` to restart containers with %s...\n", dockerCmd, yaml)
	CheckYamlExists(yaml)
	startErr := RunCmd(dockerCmd, []string{"-f", yaml, "restart"})
	if startErr != nil {
		log.Fatalf("Error trying to restart the containers with %s: %v\n", yaml, startErr)
	}
}

// RunDockerComposeUp brings up Docker containers in detached mode using the specified Docker Compose YAML file.
// Exits fatally if the YAML file does not exist or if the command fails.
func RunDockerComposeUp(yaml string) {
	fmt.Printf("[+] Running `%s` to bring up the containers with %s...\n", dockerCmd, yaml)
	CheckYamlExists(yaml)
	upErr := RunCmd(dockerCmd, []string{"-f", yaml, "up", "-d"})
	if upErr != nil {
		log.Fatalf("Error trying to bring up the containers with %s: %v\n", yaml, upErr)
	}
}

// RunDockerComposeDown stops and removes containers defined in the specified Docker Compose YAML file.
// If volumes is true, associated Docker volumes are also removed. Exits fatally on failure.
func RunDockerComposeDown(yaml string, volumes bool) {
	fmt.Printf("[+] Running `%s` to bring down the containers with %s...\n", dockerCmd, yaml)
	args := []string{"-f", yaml, "down"}
	if volumes {
		args = append(args, "--volumes")
	}
	CheckYamlExists(yaml)
	downErr := RunCmd(dockerCmd, args)
	if downErr != nil {
		log.Fatalf("Error trying to bring down the containers with %s: %v\n", yaml, downErr)
	}
}

// RunDockerComposePull pulls the latest container images defined in the specified Docker Compose YAML file.
// Exits fatally if the YAML file does not exist or if the pull operation fails.
func RunDockerComposePull(yaml string) {
	fmt.Printf("[+] Running `%s` to pull container images with %s...\n", dockerCmd, yaml)
	CheckYamlExists(yaml)
	startErr := RunCmd(dockerCmd, []string{"-f", yaml, "pull"})
	if startErr != nil {
		log.Fatalf("Error trying to pull the container images with %s: %v\n", yaml, startErr)
	}
}

// FetchLogs fetches logs from the container with the specified "name" label ("containerName" parameter).
func FetchLogs(containerName string, lines string) []string {
	var logs []string
	cli, err := client.New(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Failed to get client in logs: %v", err)
	}
	containers, err := cli.ContainerList(context.Background(), client.ContainerListOptions{})
	if err != nil {
		log.Fatalf("Failed to get container list: %v", err)
	}
	if len(containers.Items) > 0 {
		for _, container := range containers.Items {
			if container.Labels["name"] == containerName || containerName == "all" || container.Labels["name"] == "bhce_"+containerName {
				logs = append(logs, fmt.Sprintf("\n*** Logs for `%s` ***\n\n", container.Labels["name"]))
				reader, err := cli.ContainerLogs(context.Background(), container.ID, client.ContainerLogsOptions{
					ShowStdout: true,
					ShowStderr: true,
					Tail:       lines,
				})
				if err != nil {
					log.Fatalf("Failed to get container logs: %v", err)
				}
				defer reader.Close()
				// Reference: https://medium.com/@dhanushgopinath/reading-docker-container-logs-with-golang-docker-engine-api-702233fac044
				p := make([]byte, 8)
				_, err = reader.Read(p)
				for err == nil {
					content := make([]byte, binary.BigEndian.Uint32(p[4:]))
					reader.Read(content)
					logs = append(logs, string(content))
					_, err = reader.Read(p)
				}
			}
		}

		if len(logs) == 0 {
			logs = append(logs, fmt.Sprintf("\n*** No logs found for requested container '%s' ***\n", containerName))
		}
	} else {
		fmt.Println("Failed to find that container")
	}
	return logs
}

// GetRunning determines if the container with the specified "name" label ("containerName" parameter) is running.
func GetRunning() Containers {
	var running Containers

	cli, err := client.New(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Failed to get client connection to Docker: %v", err)
	}
	containers, err := cli.ContainerList(context.Background(), client.ContainerListOptions{
		All: false,
	})
	if err != nil {
		log.Fatalf("Failed to get container list from Docker: %v", err)
	}
	if len(containers.Items) > 0 {
		for _, container := range containers.Items {
			if Contains(devImages, container.Labels["name"]) || Contains(prodImages, container.Labels["name"]) {
				running = append(running, Container{
					container.ID, container.Image, container.Status, container.Ports, container.Labels["name"],
				})
			}
		}
	}

	return running
}

// ResetAdminPassword executes the "docker compose" commands to brings containers down and back up to reset the default
// admin account for the specified YAML file ("yaml" parameter).
func ResetAdminPassword(yaml string) {
	RunDockerComposeDown(yaml, false)
	bhEnv.Set("default_admin.password", GenerateRandomPassword(32, true))
	WriteBloodHoundEnvironmentVariables()
	envErr := os.Setenv("bhe_recreate_default_admin", "true")
	if envErr != nil {
		log.Fatalf("Error setting the necessary `bhe_recreate_default_admin` environment variable: %v\n", envErr)
	}
	RunDockerComposeUp(yaml)
	fmt.Println("[+] BloodHound is ready to go!")
	fmt.Printf("[+] You can log in as `%s` with this password: %s\n", bhEnv.GetString("default_admin.principal_name"), bhEnv.GetString("default_admin.password"))
	fmt.Println("[+] You can get your admin password by running: bloodhound-cli config get default_password")
	fmt.Printf("[+] You can access the BloodHound UI at: %s%s\n", bhEnv.GetString("root_url"), loginUri)
}
