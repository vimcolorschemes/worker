package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vimcolorschemes/worker/internal/color"
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
func Generate(force bool, repoKey string) bson.M {
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
	} else {
		repositories = database.GetValidRepositories()
	}

	log.Printf("Generating vim preview for %d repositories", len(repositories))

	var generateCount int

	for _, repository := range repositories {
		fmt.Println()

		log.Print("Generating vim previews for ", repository.Owner.Name, "/", repository.Name)

		if !force && repository.GeneratedAt.After(repository.LastCommitAt) {
			log.Print("Repository is not due for a generate")
			continue
		}

		generateCount++

		newVimColorSchemes := repository.VimColorSchemes

		pluginPath := fmt.Sprintf("colors/%s__%s", repository.Owner.Name, repository.Name)
		err := installPlugin(repository.GitHubURL, pluginPath)
		if err != nil {
			log.Print(err)
			continue
		}

		for index, vimColorScheme := range newVimColorSchemes {
			file.DownloadFile(vimColorScheme.FileURL, fmt.Sprintf("%s/colors/%s.vim", tmpDirectoryPath, vimColorScheme.Name))

			lightVimColorSchemeColors, err := getVimColorSchemeColorData(vimColorScheme, repoHelper.LightBackground)
			if err != nil {
				log.Print(err)
				continue
			}

			darkVimColorSchemeColors, err := getVimColorSchemeColorData(vimColorScheme, repoHelper.DarkBackground)
			if err != nil {
				log.Print(err)
				continue
			}

			vimColorSchemeData := repoHelper.VimColorSchemeData{
				Light: lightVimColorSchemeColors,
				Dark:  darkVimColorSchemeColors,
			}

			newVimColorSchemes[index] = repoHelper.VimColorScheme{
				Name:            vimColorScheme.Name,
				FileURL:         vimColorScheme.FileURL,
				Data:            vimColorSchemeData,
				Valid:           true,
				UsesXtermColors: vimColorScheme.UsesXtermColors,
			}
		}

		repository.VimColorSchemes = newVimColorSchemes
		repository.GenerateValid = repoHelper.IsRepositoryValidAfterGenerate(repository)

		generateObject := getGenerateRepositoryObject(repository)
		database.UpsertRepository(repository.ID, generateObject)
	}

	cleanUp()

	return bson.M{"repositoryCount": generateCount}
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

	runtimepath := fmt.Sprintf("let &runtimepath.=',%s/colors'\n\n", tmpDirectoryPath)

	vimrcContent := fmt.Sprintf("%s\n%s\n%s", baseVimrcContent, myVimrc, runtimepath)

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

	err = addSubdirectoriesToRuntimepath(target)
	if err != nil {
		return err
	}

	return nil
}

func addSubdirectoriesToRuntimepath(path string) error {
	var paths []string

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() || strings.Contains(path, ".git") {
			return nil
		}

		paths = append(paths, path)

		return nil
	})
	if err != nil {
		return err
	}

	runtimepath := fmt.Sprintf("let &runtimepath.=',%s'\n\n", strings.Join(paths, ","))

	err = file.AppendToFile(runtimepath, vimrcPath)
	if err != nil {
		return err
	}

	return nil
}

// Gathers the color scheme data on a specific background from vcspg.vim
func getVimColorSchemeColorData(vimColorScheme repoHelper.VimColorScheme, background repoHelper.VimBackgroundValue) ([]repoHelper.VimColorSchemeGroup, error) {
	err := executePreviewGenerator(vimColorScheme, background)
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

	var vimColorSchemeColorsResult map[string]string
	err = json.Unmarshal([]byte(vimColorSchemeOutput), &vimColorSchemeColorsResult)
	if err != nil {
		return nil, err
	}

	vimColorSchemeColors := make([]repoHelper.VimColorSchemeGroup, 0, len(vimColorSchemeColorsResult))

	for groupName, colorCode := range vimColorSchemeColorsResult {
		if vimColorScheme.UsesXtermColors {
			colorCode = color.ConvertXtermToHexCode(colorCode)
		}

		vimColorSchemeColors = append(vimColorSchemeColors, repoHelper.VimColorSchemeGroup{
			Name:    groupName,
			HexCode: colorCode,
		})
	}

	return vimColorSchemeColors, nil
}

// Starts a vim instance and auto commands to configure and start vcspg.vim on load
func executePreviewGenerator(vimColorScheme repoHelper.VimColorScheme, background repoHelper.VimBackgroundValue) error {
	writeColorValuesAutoCmd := fmt.Sprintf("autocmd ColorScheme * :call WriteColorValues(\"%s/data.json\",\"%s\")", tmpDirectoryPath, background)
	setBackground := fmt.Sprintf("set background=%s", background)
	setColorScheme := fmt.Sprintf("silent! colorscheme %s", vimColorScheme.Name)

	args := []string{
		"-u", vimrcPath,
		"-c", writeColorValuesAutoCmd,
	}

	if !vimColorScheme.UsesXtermColors {
		args = append(args, "-c set termguicolors")
	}

	args = append(args,
		"-c", setBackground,
		"-c", setColorScheme,
		"-c", ":qa!",
		"./vim/code_sample.vim",
	)

	cmd := exec.Command("vim", args...)

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
		"generateValid":   repository.GenerateValid,
		"generatedAt":     time.Now(),
	}
}
