// NCAA Checkver (version Barcelona-0.0.1)
// by Matt Stills <matthew.stills@turner.com>
//
// Used to quickly determine the current version of a module by consulting
// the site makefile.
//
// You may set the following environment variables to avoid having to
// pass options for these values each time you use the utility:
//
// NCAA_BARCA_SITE_REPO_PATH (default = "~/Repos/ncaa-barcelona")
// NCAA_BARCA_SITE_MAKEFILE  (default = "barcelona.make")
package main

import (
    "bufio"
    "flag"
    "fmt"
    "io/ioutil"
    "os"
    "os/exec"
    "os/user"
    "regexp"
    "strings"
)

type nestedMap map[string]map[string]string
type cvError struct {
    msg string
}

var (
    // options for this utility
    siteRepoOpt   string
    siteMakeOpt   string
    siteBranchOpt string
    // location of the makefile and module
    makefile string
    module   string
)

var usr, _ = user.Current()
var optionsMap = nestedMap{
    "site-repo": {
        "usage":   "The path to your site (app) repo where the makefile resides.",
        "default": usr.HomeDir + "/Repos/ncaa-barcelona",
    },
    "site-makefile": {
        "usage":   "Filename of the *.make file to alter.",
        "default": "barcelona.make",
    },
    "site-branch": {
        "usage":   "Branch of the site repo to check version in (dev|qa|master)",
        "default": "master",
    },
}

// error reporter/handler for the utility
func (e *cvError) Error() string {
    return fmt.Sprintf("%s", e.msg)
}

// applyEnvOptions checks whether or not site options are using defaults. If so,
// it attemps to use environment variables to set them instead.
func applyEnvOptions() {
    if siteRepoOpt == optionsMap["site-repo"]["default"] {
        if envRepo := os.Getenv("NCAA_BARCA_SITE_REPO_PATH"); envRepo != "" {
            siteRepoOpt = envRepo
        }
    }

    if siteMakeOpt == optionsMap["site-makefile"]["default"] {
        if envMake := os.Getenv("NCAA_BARCA_SITE_MAKEFILE"); envMake != "" {
            siteMakeOpt = envMake
        }
    }

    if siteMakeOpt == optionsMap["site-branch"]["default"] {
        if envMake := os.Getenv("NCAA_BARCA_SITE_BRANCH"); envMake != "" {
            siteMakeOpt = envMake
        }
    }
}

// git runs a git command in given directory
func git(command []string, dir string) []byte {
    os.Chdir(dir)
    out, err := exec.Command("git", command...).CombinedOutput()

    if err != nil {
        fmt.Println(string(out))
        panic("there was a problem running the git command '" + strings.Join(command, " ") + "'. See output above for clues.")
    }

    return out
}

// getMakefile reads the provided site directory and locates the makefile
func getMakefile() (string, error) {
    var makefile string

    siteFiles, err := ioutil.ReadDir(siteRepoOpt)
    foundMakefile := false

    if err != nil {
        return "", &cvError{("There was a problem reading the site repo directory @ " + siteRepoOpt)}
    }

    for _, file := range siteFiles {
        if file.Name() == siteMakeOpt {
            foundMakefile = true
            break
        }
    }

    if !foundMakefile {
        return "", &cvError{("Could not locate makefile @ '" + siteRepoOpt + "/" + siteMakeOpt + "'")}
    }

    makefile = siteRepoOpt + "/" + siteMakeOpt

    return makefile, nil
}

func init() {
    // option: --site-repo / -r
    flag.StringVar(&siteRepoOpt, "site-repo", optionsMap["site-repo"]["default"], optionsMap["site-repo"]["usage"])
    flag.StringVar(&siteRepoOpt, "r", optionsMap["site-repo"]["default"], "shorthand for --site-repo")

    // option: --site-makefile
    flag.StringVar(&siteMakeOpt, "site-makefile", optionsMap["site-makefile"]["default"], optionsMap["site-makefile"]["usage"])

    // option: --site-branch / -b
    flag.StringVar(&siteBranchOpt, "site-branch", optionsMap["site-branch"]["default"], optionsMap["site-branch"]["usage"])
    flag.StringVar(&siteBranchOpt, "b", optionsMap["site-branch"]["default"], "shorthand for --site-branch")
}

func main() {
    var err error

    // ** parse flags, apply env options, extract module arg
    flag.Parse()
    applyEnvOptions()

    if args := flag.Args(); len(args) > 0 {
        module = args[0:1][0]
    }

    // ** determine current branch so we can go back to it later
    originalBranch := string(git([]string{"rev-parse", "--abbrev-ref", "HEAD"}, siteRepoOpt))
    originalBranch = strings.Trim(originalBranch, " \n\t\r")

    git([]string{"checkout", ("origin/" + siteBranchOpt)}, siteRepoOpt)

    makefile, err = getMakefile()

    if err != nil {
        panic(err)
    }

    // ** scan makefile for module
    var matches [][]string

    file, _ := os.Open(makefile)
    defer file.Close()

    scanner := bufio.NewScanner(file)
    seekLine := ("projects[" + module + "][download]")
    pattern, _ := regexp.Compile("\\[download\\]\\[(branch|tag)\\] \\= \"([A-Za-z0-9.-_]+)\"")

    // read the makefile in line by line using the scanner
    for scanner.Scan() {
        if strings.Contains(scanner.Text(), seekLine) {
            matches = pattern.FindAllStringSubmatch(scanner.Text(), -1)
        }
    }

    if (len(matches) > 0 && len(matches[0]) == 3) {
        fmt.Printf("[%s] Current %s: %s\n", module, matches[0][1], matches[0][2])
    } else {
        fmt.Println("Module not found:", module)
    }

    git([]string{"checkout", originalBranch}, siteRepoOpt)
}