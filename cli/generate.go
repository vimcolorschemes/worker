package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/vimcolorschemes/worker/internal/database"
	file "github.com/vimcolorschemes/worker/internal/file"
	repoHelper "github.com/vimcolorschemes/worker/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
)

var tmpDirectoryPath string
var vimrcPath string
var vimFilesPath string
var colorDataFilePath string

// Generate vim color scheme data for all valid repositories
func Generate(force bool) {
	log.Print("Running generate")

	startTime := time.Now()

	fmt.Println()

	initVimFiles()

	setupVim()

	fmt.Println()

	repositories := database.GetValidRepositories()

	log.Printf("Generating vim preview for %d repositories", len(repositories))

	for _, repository := range repositories {
		fmt.Println()

		log.Print("Generating vim previews for ", repository.Owner.Name, "/", repository.Name)

		if !force && repository.GeneratedAt.After(repository.LastCommitAt) {
			log.Print("Repository is not due for a generate")
			continue
		}

		newVimColorSchemes := repository.VimColorSchemes

		pluginPath := fmt.Sprintf("colors/%s__%s", repository.Owner.Name, repository.Name)
		err := installPlugin(repository.GitHubURL, pluginPath)
		if err != nil {
			log.Print(err)
			continue
		}

		for index, vimColorScheme := range newVimColorSchemes {
			lightVimColorSchemeColors, err := getVimColorSchemeColorData(vimColorScheme.Name, repoHelper.LightBackground)
			if err != nil {
				log.Print(err)
				continue
			}

			darkVimColorSchemeColors, err := getVimColorSchemeColorData(vimColorScheme.Name, repoHelper.DarkBackground)
			if err != nil {
				log.Print(err)
				continue
			}

			vimColorSchemeData := repoHelper.VimColorSchemeData{
				Light: lightVimColorSchemeColors,
				Dark:  darkVimColorSchemeColors,
			}

			newVimColorSchemes[index] = repoHelper.VimColorScheme{
				Name:    vimColorScheme.Name,
				FileURL: vimColorScheme.FileURL,
				Data:    vimColorSchemeData,
				Valid:   true,
			}
		}

		repository.VimColorSchemes = newVimColorSchemes
		generateObject := getGenerateRepositoryObject(repository)
		database.UpsertRepository(repository.ID, generateObject)
	}

	fmt.Println()

	database.CreateReport("generate", time.Since(startTime).Seconds(), bson.M{})

	fmt.Println()

	log.Print(":wq")

	cleanUp()
}

// Initializes a temporary directory for vim configuration files
func initVimFiles() {
	workingDirectory, err := os.Getwd()
	if err != nil {
		log.Panic(err)
	}

	tmpDirectoryPath = fmt.Sprintf("%s/.tmp", workingDirectory)
	vimFilesPath = fmt.Sprintf("%s/vim", workingDirectory)
	vimrcPath = fmt.Sprintf("%s/.vimrc", tmpDirectoryPath)
	colorDataFilePath = fmt.Sprintf("%s/data.json", tmpDirectoryPath)

	if _, err := os.Stat(tmpDirectoryPath); !os.IsNotExist(err) {
		// .tmp directory exists, remove it
		err := os.RemoveAll(tmpDirectoryPath)
		if err != nil {
			log.Panic(err)
		}
	}

	log.Printf("Creating tmp directory: %s", tmpDirectoryPath)
	err = os.Mkdir(tmpDirectoryPath, os.FileMode(0700))
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Creating tmp .vimrc: %s", vimrcPath)
	_, err = os.Create(vimrcPath)
	if err != nil {
		log.Panic(err)
	}
}

// Sets up the vim configuration common to all vim color schemes
func setupVim() {
	log.Print("Setting up vim config")

	baseVimrcContent, err := file.GetLocalFileContent(fmt.Sprintf("%s/base_vimrc.vim", vimFilesPath))
	if err != nil {
		log.Panic(err)
	}

	myVimrc := fmt.Sprintf("let $MYVIMRC='%s'\n\n", vimrcPath)

	vimrcContent := fmt.Sprintf("%s\n%s", baseVimrcContent, myVimrc)

	err = file.AppendToFile(vimrcContent, vimrcPath)
	if err != nil {
		log.Panic(err)
	}

	vcspg, err := file.GetLocalFileContent(fmt.Sprintf("%s/vcspg.vim", vimFilesPath))
	if err != nil {
		log.Panic(err)
	}

	err = file.AppendToFile(vcspg, vimrcPath)
	if err != nil {
		log.Panic(err)
	}

	err = installPlugin("https://github.com/sheerun/vim-polyglot", "plugins/vim-polyglot")
	if err != nil {
		log.Panic(err)
	}
}

// Installs a plugin/color scheme on the vim configuration from a GitHub URL
func installPlugin(gitRepositoryURL string, path string) error {
	log.Printf("Installing %s", path)

	target := fmt.Sprintf("%s/%s", tmpDirectoryPath, path)
	err := os.MkdirAll(target, 0700)
	if err != nil {
		return err
	}

	cmd := exec.Command("git", "clone", gitRepositoryURL, target)
	err = cmd.Run()
	if err != nil {
		return err
	}

	runtimePath := fmt.Sprintf("let &runtimepath.=',%s'\n\n", target)
	err = file.AppendToFile(runtimePath, vimrcPath)
	if err != nil {
		return err
	}

	return nil
}

// Gathers the color scheme data on a specific background from vcspg.vim
func getVimColorSchemeColorData(colorSchemeName string, background repoHelper.VimBackgroundValue) (repoHelper.VimColorSchemeColorDefinitions, error) {
	err := executePreviewGenerator(colorSchemeName, background)
	if err != nil {
		return nil, err
	}

	vimColorSchemeOutput, err := file.GetLocalFileContent(colorDataFilePath)
	if err != nil {
		return nil, err
	}

	err = os.Remove(colorDataFilePath)
	if err != nil {
		return nil, err
	}

	var vimColorSchemeColors repoHelper.VimColorSchemeColorDefinitions
	err = json.Unmarshal([]byte(vimColorSchemeOutput), &vimColorSchemeColors)
	if err != nil {
		return nil, err
	}

	if len(vimColorSchemeColors) == 0 {
		// Store nil instead of empty object if no data was gathered
		return nil, nil
	}

	return vimColorSchemeColors, nil
}

// Starts a vim instance and auto commands to configure and start vcspg.vim on load
func executePreviewGenerator(colorSchemeName string, background repoHelper.VimBackgroundValue) error {
	writeColorValuesAutoCmd := fmt.Sprintf("autocmd ColorScheme * :call WriteColorValues(\"%s/data.json\",\"%s\")", tmpDirectoryPath, background)
	setBackground := fmt.Sprintf("set background=%s", background)
	setColorScheme := fmt.Sprintf("silent! colorscheme %s", colorSchemeName)

	cmd := exec.Command(
		"vim",
		"-u", vimrcPath,
		"-c", writeColorValuesAutoCmd,
		"-c", setBackground,
		"-c", setColorScheme,
		"-c", ":qa!",
		"./vim/code_sample.vim",
	)

	log.Printf("Running %s", cmd)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

// Deletes the temporary directory used for the vim config
func cleanUp() {
	err := os.RemoveAll(tmpDirectoryPath)
	if err != nil {
		log.Panic(err)
	}
}

func getGenerateRepositoryObject(repository repoHelper.Repository) bson.M {
	return bson.M{
		"vimColorSchemes": repository.VimColorSchemes,
		"valid":           repository.Valid,
		"generatedAt":     time.Now(),
	}
}
