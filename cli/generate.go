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
)

var tmpDirectoryPath string
var packDirectoryPath string
var vimrcPath string
var vimFilesPath string
var colorDataFilePath string
var defaultColorschemeFilePath string
var defaultColorschemes map[string]bool
var debugMode bool

// Generate color scheme data for all valid repositories
func Generate(force bool, debug bool, repoKey string) map[string]interface{} {
	debugMode = debug

	initRuntimeFiles()

	setupRuntime()

	fmt.Println()

	var repositories []repoHelper.Repository
	if repoKey != "" {
		repository, err := database.GetRepository(repoKey)
		if err != nil {
			log.Panic(err)
		}
		repositories = []repoHelper.Repository{repository}
	} else if force || debug {
		var err error
		repositories, err = database.GetRepositories()
		if err != nil {
			log.Panic(err)
		}
	} else {
		var err error
		repositories, err = database.GetRepositoriesToGenerate()
		if err != nil {
			log.Panic(err)
		}
	}

	log.Printf("Generating previews for %d repositories", len(repositories))

	for index, repository := range repositories {
		log.Print("\nGenerating previews for ", repository.Owner.Name, "/", repository.Name, " (", index+1, "/", len(repositories), ")")

		key := fmt.Sprintf("%s__%s", repository.Owner.Name, repository.Name)
		err := installPlugin(repository.GithubURL, key)
		if err != nil {
			log.Printf("Error installing plugin: %s", err)
			updateRepositoryAfterGenerate(repository)
			continue
		}

		var data, dataError = getColorSchemeColorData()
		err = deletePlugin(key)
		if err != nil {
			log.Printf("Error deleting plugin: %s", err)
		}
		if dataError != nil {
			log.Printf("Error getting color data: %s", dataError)
			updateRepositoryAfterGenerate(repository)
			continue
		}

		var colorSchemes []repoHelper.ColorScheme

		for name := range data {
			// Skip built-in colorschemes
			if defaultColorschemes[name] || isDefaultColorscheme(name) {
				continue
			}

			var backgrounds []repoHelper.BackgroundValue
			if data[name].Light != nil {
				backgrounds = append(backgrounds, repoHelper.LightBackground)
			}
			if data[name].Dark != nil {
				backgrounds = append(backgrounds, repoHelper.DarkBackground)
			}

			colorSchemes = append(
				colorSchemes,
				repoHelper.ColorScheme{
					Name:        name,
					Data:        data[name],
					Backgrounds: backgrounds,
				})
		}

		repository.ColorSchemes = colorSchemes
		updateRepositoryAfterGenerate(repository)
	}

	cleanUp()

	return map[string]interface{}{"repositoryCount": len(repositories)}
}

func updateRepositoryAfterGenerate(repository repoHelper.Repository) {
	log.Printf("Generated %d color schemes", len(repository.ColorSchemes))
	data := getGenerateData(repository)
	database.UpdateRepositoryFromGenerate(repository.ID, data)
}

// Initializes a temporary directory for runtime configuration files
func initRuntimeFiles() {
	workingDirectory, err := os.Getwd()
	if err != nil {
		log.Panic(err)
	}

	tmpDirectoryPath = fmt.Sprintf("%s/.tmp", workingDirectory)
	packDirectoryPath = fmt.Sprintf("%s/pack/plugins/start", tmpDirectoryPath)
	vimFilesPath = fmt.Sprintf("%s/vim", workingDirectory)
	vimrcPath = fmt.Sprintf("%s/init.lua", tmpDirectoryPath)
	colorDataFilePath = fmt.Sprintf("%s/data.json", tmpDirectoryPath)
	defaultColorschemeFilePath = fmt.Sprintf("%s/default_colorschemes.json", tmpDirectoryPath)

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

// Sets up the runtime configuration common to all color schemes
func setupRuntime() {
	log.Print("Setting up runtime config")

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

	captureDefaultColorschemes()
}

// captureDefaultColorschemes runs nvim to get the list of built-in colorschemes
// and populates the defaultColorschemes map.
func captureDefaultColorschemes() {
	log.Print("Capturing default colorschemes")

	cmd := exec.Command("nvim", "-u", vimrcPath, "--headless",
		"-c", fmt.Sprintf("lua require('extractor').colorschemes({ output_path = '%s' })", defaultColorschemeFilePath),
		"-c", "qa!")

	log.Printf("Running %s", cmd)
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	if err != nil {
		log.Panic(err)
	}

	content, err := file.GetLocalFileContent(defaultColorschemeFilePath)
	if err != nil {
		log.Panic(err)
	}

	var names []string
	err = json.Unmarshal([]byte(content), &names)
	if err != nil {
		log.Panic(err)
	}

	defaultColorschemes = make(map[string]bool, len(names))
	for _, name := range names {
		defaultColorschemes[name] = true
	}

	log.Printf("Captured %d default colorschemes", len(defaultColorschemes))
}

// Installs a plugin/color scheme on the runtime configuration from a Github URL
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

// Clears all installation traces of the plugin
func deletePlugin(key string) error {
	// Remove downloaded files
	target := fmt.Sprintf("%s/%s", packDirectoryPath, key)
	err := os.RemoveAll(target)
	return err
}

// Gathers the colorscheme data from vimcolorschemes/extractor.nvim
func getColorSchemeColorData() (map[string]repoHelper.ColorSchemeData, error) {
	err := executePreviewGenerator()
	if err != nil {
		log.Printf("Error executing nvim: %s", err)
		return nil, err
	}

	colorSchemeOutput, err := file.GetLocalFileContent(colorDataFilePath)
	if err != nil {
		log.Printf("Error getting local file content from \"%s\": %s", colorDataFilePath, err)
		return nil, err
	}

	var data map[string]repoHelper.ColorSchemeData
	err = json.Unmarshal([]byte(colorSchemeOutput), &data)
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

// Starts a runtime instance and auto commands to configure and start vcspg on load
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

// Deletes the temporary directory used for runtime config
func cleanUp() {
	if debugMode {
		return
	}

	err := os.RemoveAll(tmpDirectoryPath)
	if err != nil {
		log.Panic(err)
	}
}

func getGenerateData(repository repoHelper.Repository) database.GenerateData {
	return database.GenerateData{
		ColorSchemes: repository.ColorSchemes,
		GeneratedAt:  time.Now(),
	}
}

func isDefaultColorscheme(name string) bool {
	defaultNames := map[string]bool{
		"default":  true,
		"habamax":  true,
		"slate":    true,
		"zaibatsu": true,
	}

	return defaultNames[name]
}
