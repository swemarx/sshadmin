package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/go-ini/ini"
	"github.com/pborman/getopt/v2"
	"os"
	"os/exec"
	"sync"
)

var (
	userCommand string                      // Mandatory
	username    string                      // Mandatory
	hosts       []string                    // Mandatory
	iniFile     string   = "/.sa/hosts.ini" // Optional below
	DEBUG       bool     = false
	runInSeq    bool     = false
	doPrefix    bool     = false
	yesMode     bool     = false
)

func getCommandLineArgs() []string {
	getopt.FlagLong(&iniFile, "inifile", 'f', "Ini-file containing host/group definitions")
	getopt.FlagLong(&userCommand, "command", 'c', "Command to run")
	getopt.FlagLong(&username, "username", 'u', "User to connect as")
	getopt.FlagLong(&runInSeq, "sequence", 's', "Run in sequence in stead of in parallel")
	getopt.FlagLong(&doPrefix, "prefix", 'p', "Prefix each line of output with hostname")
	getopt.FlagLong(&DEBUG, "debug", 'd', "Debugmode")
	getopt.FlagLong(&yesMode, "yes", 'y', "Assumes yes on questions")
	getopt.Parse()

	if !getopt.IsSet("command") || !getopt.IsSet("username") {
		getopt.PrintUsage(os.Stdout)
		fmt.Println("\n[error] you need to specify command/username/hosts!")
		os.Exit(1)
	}

	if DEBUG {
		fmt.Printf("[debug] using configuration %s\n", iniFile)
	}

	if len(getopt.Args()) == 0 {
		fmt.Println("[error] no hosts specified")
		os.Exit(1)
	}
	return getopt.Args()
}

func loadIni(filePath string) *ini.File {
	if !getopt.IsSet("inifile") {
		iniFile = os.Getenv("HOME") + iniFile
	}
	f, err := ini.LoadSources(ini.LoadOptions{AllowBooleanKeys: true, Insensitive: true}, iniFile)
	if err != nil {
		fmt.Println("[error] Could not load", filePath)
		os.Exit(1)
	}
	return f
}

func removeDuplicates(a []string) []string {
	result := []string{}
	seen := map[string]string{}
	for _, val := range a {
		if _, ok := seen[val]; !ok {
			result = append(result, val)
			seen[val] = val
		}
	}
	return result
}

func resolveHosts(cfg *ini.File, args []string) []string {
	var h []string

	for _, arg := range args {
		var (
			isHost    bool = false
			isSection bool = false
		)

		if DEBUG {
			fmt.Printf("[debug] resolving \"%s\": ", arg)
		}

		/* does not work atm
		// is "arg" a key?
		if cfg.Section("").HasKey(arg) {
			fmt.Println("HOST")
			h = append(h, arg)
			continue
		}
		*/

		/* works, but replaced with one scan for both sections and hosts.
		// is "arg" a section?
		if sec, _ := cfg.GetSection(arg); sec != nil {
			fmt.Println("SECTION")
			fmt.Printf("%+v\n", sec.KeyStrings())
			//h = append(h, sec.KeyStrings())
			continue
		}
		*/

		for _, section := range cfg.Sections() {
			if section.Name() == arg {
				isSection = true
				h = append(h, section.KeyStrings()...)
			}
			for _, key := range section.KeyStrings() {
				if key == arg {
					isHost = true
					h = append(h, arg)
				}
			}
		}

		if isSection && isHost {
			fmt.Printf("[error] \"%s\" is both section and host, exiting\n", arg)
			os.Exit(1)
		} else if isSection {
			if DEBUG {
				fmt.Println("section")
			}
		} else if isHost {
			if DEBUG {
				fmt.Println("host")
			}
		} else {
			fmt.Printf("[error] could not resolve \"%s\"\n", arg)
			os.Exit(1)
		}
	}
	//sort.Strings(h)
	return removeDuplicates(h)
}

// Returns true if yes, false if no
func askUser(message string) bool {
	fmt.Printf(message)
	r := bufio.NewReader(os.Stdin)
	i, _ := r.ReadString('\n')
	switch i[0:1] {
	case "y":
		return true
	case "n":
		return false
	default:
		return false
	}
}

func runCommand(u string, h string, c string, wg *sync.WaitGroup) {
	if !runInSeq {
		defer wg.Done()
	}
	cmd := exec.Command("ssh", u+"@"+h, c)
	o, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
	}
	if doPrefix {
		p := "[" + h + "] "
		as := bytes.SplitAfter(o, []byte("\n"))
		n := len(as)
		if DEBUG {
			fmt.Printf("[debug] output-array contains %d elements\n", n)
		}
		for i := 0; i < n; i++ {
			asl := len(as[i])
			if asl == 0 {
				continue
			}
			// Some lines do not end with "\n"
			if as[i][asl-1] != '\n' {
				fmt.Printf("%s%s\n", p, string(as[i]))
			} else {
				fmt.Printf("%s%s", p, string(as[i]))
			}
		}
	} else {
		fmt.Printf("%s", o)
	}
}

func main() {
	var wg sync.WaitGroup

	args := getCommandLineArgs()
	if DEBUG {
		fmt.Printf("[debug] remaining args: %s\n", args)
	}
	cfg := loadIni(iniFile)
	hosts := resolveHosts(cfg, args)
	fmt.Printf("Running command \"%s\" on %d hosts: %s\n", userCommand, len(hosts), hosts)
	if !yesMode {
		if !askUser("Continue? [yes/no] ") {
			fmt.Println("Cancelled.. exiting")
			os.Exit(1)
		}
	}

	for _, host := range hosts {
		if runInSeq {
			runCommand(username, host, userCommand, nil)
		} else {
			wg.Add(1)
			go runCommand(username, host, userCommand, &wg)
		}
	}
	if !runInSeq {
		wg.Wait()
	}
}
