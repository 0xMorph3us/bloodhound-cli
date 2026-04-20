package cmd

import (
	"fmt"
	"github.com/0xMorph3us/bloodhound-cli/cmd/config"
	"github.com/spf13/cobra"
	"os"
	"text/tabwriter"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Displays BloodHound CLI's version information",
	Long: `Displays BloodHound CLI's local version information from the current binary.`,
	RunE: compareCliVersions,
}

// init registers the version command with the root command, enabling the "version" CLI command.
func init() {
	rootCmd.AddCommand(versionCmd)
}

// compareCliVersions prints BloodHound CLI local version numbers and build dates from
// the current binary without calling remote release APIs.
func compareCliVersions(cmd *cobra.Command, args []string) error {
	// initialize tabwriter
	writer := new(tabwriter.Writer)
	// Set minwidth, tabwidth, padding, padchar, and flags
	writer.Init(os.Stdout, 8, 8, 1, '\t', 0)

	defer writer.Flush()

	fmt.Println("[+] Displaying local version information:")

	if len(config.BuildDate) == 0 {
		fmt.Fprintf(writer, "\nLocal Version\tBloodHound CLI %s", config.Version)
	} else {
		fmt.Fprintf(writer, "\nLocal Version\tBloodHound CLI %s (%s)", config.Version, config.BuildDate)
	}

	fmt.Fprintf(writer, "\nUpdate Method\tBuild from source (`git pull && make`)\n")

	return nil
}
