package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"gopkg.in/xmlpath.v1"
)

var commands []string

type Miseq interface {
	appendCommands(string) (string, error)
}

type MiseqRun struct {
	xpath   string
	command string
	regex   string
	Miseq
}

type BlackjackRun struct {
	xpath   string
	command string
	regex   string
}

func (run BlackjackRun) appendCommands(path string) (string, error) {

	flowcellCommand := fmt.Sprintf("%s run=%s", run.command, path)
	commands = append(commands, flowcellCommand)

	return "", nil

}

type AbacusRun struct {
	xpath   string
	command string
	regex   string
}

func (run AbacusRun) appendCommands(path string) (string, error) {
	subFolders := strings.Split(path, "/")
	lastSubFolder := subFolders[len(subFolders)-1]

	myExp := regexp.MustCompile(`/data/instr_data/(.*?)/.*`)
	miSeqFolderName := myExp.FindStringSubmatch(path)[1]

	// Generate symlink path
	symLinkPath := fmt.Sprintf("/data/instr_data/%s/Abacus_PES/%s", miSeqFolderName, lastSubFolder)

	// Generate symlink command
	symLinkCommand := fmt.Sprintf("sudo ln -s %s %s", path, symLinkPath)
	fmt.Println(symLinkCommand)

	// Run symlink generation between blackjack and Abacus_PES folder
	out := fmt.Sprintf("%s\n", exec_command(symLinkCommand))
	fmt.Println(out)

	flowcellCommand := fmt.Sprintf("%s run=%s", run.command, symLinkPath)
	commands = append(commands, flowcellCommand)

	return "", nil

}
var miseqRunTypes map[string]MiseqRun

func InitMiseqExperimentTypes() {

	blackjack := BlackjackRun{
		xpath:   "/AnalysisJobInfo/Sheet/Header/ExperimentName",
		command: "/cm/shared/apps/blackjack/bin/flowcell",
		regex:   "^BST",
	}

	thapsia := BlackjackRun{
		xpath:   "/AnalysisJobInfo/Sheet/Header/ExperimentName",
		command: "/cm/shared/apps/thapsia/bin/flowcell",
		regex:   "^THAPSIA",
	}

	abacus := AbacusRun{
		xpath:   "/AnalysisJobInfo/Sheet/Header/InvestigatorName",
		command: "/cm/shared/apps/abacus-pipeline/abacus-backend/bin/flowcell",
		regex:   "^ABACUS",
	}

	miseqRunTypes = make(map[string]MiseqRun)
	miseqRunTypes["blackjack"] = MiseqRun{
		xpath:   blackjack.xpath,
		command: blackjack.command,
		regex:   blackjack.regex,
		Miseq:   blackjack,
	}

	miseqRunTypes["thapsia"] = MiseqRun{
		xpath:   thapsia.xpath,
		command: thapsia.command,
		regex:   thapsia.regex,
		Miseq:   thapsia,
	}
	miseqRunTypes["abacus"] = MiseqRun{
		xpath:   abacus.xpath,
		command: abacus.command,
		regex:   abacus.regex,
		Miseq:   abacus,
	}
}

// START OMIT
func isMiseqFile(info_xml_file_path string) (bool, string, error) {
	// Try to open file
	info_xml_file, err := os.Open(info_xml_file_path)
	if err != nil {
		return false, "", MyError{"File does not exist"}
	}
	// Try to parse file
	root, err := xmlpath.Parse(info_xml_file)
	if err != nil {
		return false, "", MyError{"Could not xml parse this file"}
	}
	// Check which application type this XML file belongs to (Blackjack/Abacus/Thapsia)
	for runType, runFile := range miseqRunTypes {
		if value, ok := xmlpath.MustCompile(runFile.xpath).String(root); ok {
			regex := regexp.MustCompile(runFile.regex)

			if regex.MatchString(value) {
				return true, runType, nil
			}
		}
// END OMIT

	}
	return false, "", MyError{"No match on any blackjack file types"}
}

func exec_command(command string, optionalParams ...string) []byte {

	command_arguments := strings.Split(command, " ")

	for _, options := range optionalParams {
		command_arguments = append(command_arguments, options)
	}

	cmd := exec.Command(command_arguments[0], command_arguments[1:]...)
	out, _ := cmd.Output()
	// if err != nil {
	// 	log.Fatal(command_arguments)
	// 	log.Fatal("ERROR: ", err)
	// 	log.Fatal(fmt.Sprintf("%s\n", out))
	// }

	return out

}

// MyError is an error implementation that includes a time and message.
type MyError struct {
	What string
}

func (e MyError) Error() string {
	return fmt.Sprintf("%v", e.What)
}