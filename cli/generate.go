package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/vimcolorschemes/worker/internal/database"
	file "github.com/vimcolorschemes/worker/internal/file"
	repoHelper "github.com/vimcolorschemes/worker/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
)

var tmpDirectoryPath string
var packDirectoryPath string
var vimrcPath string
var vimFilesPath string
var colorDataFilePath string
var debugMode bool

// Generate vim color scheme data for all valid repositories
func Generate(force bool, debug bool, repoKey string) bson.M {
	debugMode = debug

	initVimFiles()

	setupVim()

	fmt.Println()

	var repositories []repoHelper.Repository
	if repoKey != "" {
		repository, err := database.GetRepository(repoKey)
		if err != nil {
			log.Panic(err)
		}
		repositories = []repoHelper.Repository{repository}
	} else if force || debug {
		repositories = database.GetRepositories()
	} else {
		repositories = database.GetRepositoriesToGenerate()
	}

	log.Printf("Generating vim preview for %d repositories", len(repositories))

	for index, repository := range repositories {
		log.Print("\nGenerating vim previews for ", repository.Owner.Name, "/", repository.Name, " (", index+1, "/", len(repositories), ")")

		key := fmt.Sprintf("%s__%s", repository.Owner.Name, repository.Name)
		err := installPlugin(repository.GithubURL, key)
		if err != nil {
			log.Printf("Error installing plugin: %s", err)
			repository.GenerateValid = false
			updateRepositoryAfterGenerate(repository)
			continue
		}

		var data, dataError = getVimColorSchemeColorData()
		err = deletePlugin(key)
		if err != nil {
			log.Printf("Error deleting plugin: %s", err)
		}
		if dataError != nil {
			log.Printf("Error getting color data: %s", dataError)
			repository.GenerateValid = false
			updateRepositoryAfterGenerate(repository)
			continue
		}

		var vimColorSchemes []repoHelper.VimColorScheme

		for name := range data {
			if name == "default" || name == "module-injection" || name == "tick_tock" {
				continue
			}

			var backgrounds []repoHelper.VimBackgroundValue
			if data[name].Light != nil {
				backgrounds = append(backgrounds, repoHelper.LightBackground)
			}
			if data[name].Dark != nil {
				backgrounds = append(backgrounds, repoHelper.DarkBackground)
			}

			vimColorSchemes = append(
				vimColorSchemes,
				repoHelper.VimColorScheme{
					Name:        name,
					Data:        data[name],
					Backgrounds: backgrounds,
				})
		}

		repository.VimColorSchemes = vimColorSchemes
		repository.GenerateValid = len(repository.VimColorSchemes) > 0
		updateRepositoryAfterGenerate(repository)
	}

	cleanUp()

	return bson.M{"repositoryCount": len(repositories)}
}

func updateRepositoryAfterGenerate(repository repoHelper.Repository) {
	log.Printf("Generate valid: %v", repository.GenerateValid)
	generateObject := getGenerateRepositoryObject(repository)
	database.UpsertRepository(repository.ID, generateObject)
}

// Initializes a temporary directory for vim configuration files
func initVimFiles() {
	workingDirectory, err := os.Getwd()
	if err != nil {
		log.Panic(err)
	}

	tmpDirectoryPath = fmt.Sprintf("%s/.tmp", workingDirectory)
	packDirectoryPath = fmt.Sprintf("%s/pack/plugins/start", tmpDirectoryPath)
	vimFilesPath = fmt.Sprintf("%s/vim", workingDirectory)
	vimrcPath = fmt.Sprintf("%s/init.lua", tmpDirectoryPath)
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

	log.Printf("Creating pack directory: %s", packDirectoryPath)
	err = os.MkdirAll(packDirectoryPath, os.FileMode(0700))
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

	baseVimrcContent, err := file.GetLocalFileContent(fmt.Sprintf("%s/init.lua", vimFilesPath))
	if err != nil {
		log.Panic(err)
	}

	myVimrc := fmt.Sprintf("vim.env.MYVIMRC=\"%s\"\n", vimrcPath)

	runtimepath := fmt.Sprintf("vim.opt.runtimepath:append(\"%s\")\n", tmpDirectoryPath)
	packpath := fmt.Sprintf("vim.opt.packpath:append(\"%s\")\n", tmpDirectoryPath)

	colorDataPath := fmt.Sprintf("vim.env.COLOR_DATA_PATH=\"%s\"\n", colorDataFilePath)

	vimrcContent := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n", baseVimrcContent, myVimrc, runtimepath, packpath, colorDataPath)

	err = file.AppendToFile(vimrcContent, vimrcPath)
	if err != nil {
		log.Panic(err)
	}

	err = installPlugin("https://github.com/vimcolorschemes/extractor.nvim", "extractor.nvim")
	if err != nil {
		log.Panic(err)
	}

	err = installPlugin("https://github.com/rktjmp/lush.nvim", "lush.nvim")
	if err != nil {
		log.Panic(err)
	}

	err = removeDefaultColorschemes()
	if err != nil {
		log.Panic(err)
	}
}

// removeDefaultColorschemes removes all default colorschemes from the vim
// runtime, except for "default", which is needed to run the preview generator.
func removeDefaultColorschemes() error {
	tmpRuntimeFilePath := fmt.Sprintf("%s/runtime", tmpDirectoryPath)

	args := []string{"-es", "--headless", "-c", fmt.Sprintf("redir! > %s", tmpRuntimeFilePath), "-c", "echo $VIMRUNTIME", "-c", "redir END", "-c", "quit"}
	cmd := exec.Command("nvim", args...)

	log.Printf("Running %s", cmd)

	err := cmd.Run()
	if err != nil {
		return err
	}

	runtimePath, err := file.GetLocalFileContent(tmpRuntimeFilePath)
	if err != nil {
		return err
	}

	runtimePath = strings.TrimSpace(runtimePath)
	runtimePath = strings.Trim(runtimePath, "\n")

	colorsPath := fmt.Sprintf("%s/colors", runtimePath)
	log.Printf("Removing default colorschemes from: %s", colorsPath)

	files, err := os.ReadDir(colorsPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() || file.Name() == "default.vim" {
			continue
		}

		err = os.Remove(fmt.Sprintf("%s/%s", colorsPath, file.Name()))
		if err != nil {
			return err
		}
	}

	return nil
}

// Installs a plugin/color scheme on the vim configuration from a Github URL
func installPlugin(gitRepositoryURL string, path string) error {
	log.Printf("Installing %s", path)

	target := fmt.Sprintf("%s/%s", packDirectoryPath, path)

	cmd := exec.Command("git", "clone", gitRepositoryURL, target)
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

// Clears all installation traces of the vim plugin
func deletePlugin(key string) error {
	// Remove downloaded files
	target := fmt.Sprintf("%s/%s", packDirectoryPath, key)
	err := os.RemoveAll(target)
	return err
}

// Gathers the colorscheme data from vimcolorschemes/extractor.nvim
func getVimColorSchemeColorData() (map[string]repoHelper.VimColorSchemeData, error) {
	err := executePreviewGenerator()
	if err != nil {
		log.Printf("Error executing nvim: %s", err)
		return nil, err
	}

	vimColorSchemeOutput, err := file.GetLocalFileContent(colorDataFilePath)
	if err != nil {
		log.Printf("Error getting local file content from \"%s\": %s", colorDataFilePath, err)
		return nil, err
	}

	var data map[string]repoHelper.VimColorSchemeData
	err = json.Unmarshal([]byte(vimColorSchemeOutput), &data)
	if err != nil {
		return nil, err
	}

	if !debugMode {
		err = os.Remove(colorDataFilePath)
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}

// Starts a vim instance and auto commands to configure and start vcspg.vim on load
func executePreviewGenerator() error {
	args := []string{"-u", vimrcPath}

	if !debugMode {
		args = append(args, "--headless", "-c", ":qa!")
	}

	args = append(args, "./vim/code_sample.vim")

	cmd := exec.Command("nvim", args...)

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
	if debugMode {
		return
	}

	err := os.RemoveAll(tmpDirectoryPath)
	if err != nil {
		log.Panic(err)
	}
}

func getGenerateRepositoryObject(repository repoHelper.Repository) bson.M {
	return bson.M{
		"vimColorSchemes": repository.VimColorSchemes,
		"generateValid":   repository.GenerateValid,
		"generatedAt":     time.Now(),
	}
}
