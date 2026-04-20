package cmd

import (
	"fmt"
	docker "github.com/0xMorph3us/bloodhound-cli/cmd/internal"
	"github.com/spf13/cobra"
)

// containersRestartCmd represents the restart command
var containersRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart all stopped and running BloodHound services",
	Long: `Restart all stopped and running BloodHound services. This performs
the equivalent of running the "docker compose restart" command.`,
	Run: containersRestart,
}

func init() {
	containersCmd.AddCommand(containersRestartCmd)
}

// containersRestart restarts all BloodHound services using the Docker Compose file located in the BloodHound directory.
func containersRestart(cmd *cobra.Command, args []string) {
	docker.EvaluateDockerComposeStatus()
	fmt.Println("[+] Restarting the BloodHound environment")
	docker.RunDockerComposeRestart(docker.GetYamlFilePath(fileOverride))
}
