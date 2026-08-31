package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"time"

	"github.com/vimcolorschemes/worker/internal/database"
	file "github.com/vimcolorschemes/worker/internal/file"
	repoHelper "github.com/vimcolorschemes/worker/internal/repository"
)

const previewGenerationTimeout = 30 * time.Second

// Bounds each nvim run under previewGenerationTimeout.
const colorschemeBatchSize = 50

var tmpDirectoryPath string
var packDirectoryPath string
var vimrcPath string
var vimFilesPath string
var colorDataFilePath string
var defaultColorschemeFilePath string
var colorschemeListFilePath string
var colorschemeBatchFilePath string
var defaultColorschemes map[string]bool
var debugMode bool

// Generate colorscheme data for all valid repositories
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
	repositoryErrorCount := 0
	repositoryErrorSamples := []string{}

	for index, repository := range repositories {
		log.Print("\nGenerating previews for ", repository.Owner.Name, "/", repository.Name, " (", index+1, "/", len(repositories), ")")

		key := fmt.Sprintf("%s__%s", repository.Owner.Name, repository.Name)
		err := installPlugin(repository.GithubURL, key)
		if err != nil {
			log.Printf("Error installing plugin: %s", err)
			repositoryErrorCount++
			repositoryErrorSamples = appendRepositoryErrorSample(repositoryErrorSamples, repository, err)
			if eventErr := database.CreateRepositoryGenerateErrorEvent(repository.ID, err.Error()); eventErr != nil {
				log.Printf("Error creating generate failure event: %s", eventErr)
			}
			continue
		}

		var data, dataError = getColorschemeColorData()
		err = deletePlugin(key)
		if err != nil {
			log.Printf("Error deleting plugin: %s", err)
		}
		if dataError != nil {
			log.Printf("Error getting color data: %s", dataError)
			repositoryErrorCount++
			repositoryErrorSamples = appendRepositoryErrorSample(repositoryErrorSamples, repository, dataError)
			if eventErr := database.CreateRepositoryGenerateErrorEvent(repository.ID, dataError.Error()); eventErr != nil {
				log.Printf("Error creating generate failure event: %s", eventErr)
			}
			continue
		}

		var colorschemes []repoHelper.Colorscheme

		for name := range data {
			var backgrounds []repoHelper.BackgroundValue
			if data[name].Light != nil {
				backgrounds = append(backgrounds, repoHelper.LightBackground)
			}
			if data[name].Dark != nil {
				backgrounds = append(backgrounds, repoHelper.DarkBackground)
			}

			colorschemes = append(
				colorschemes,
				repoHelper.Colorscheme{
					Name:        name,
					Data:        data[name],
					Backgrounds: backgrounds,
				})
		}

		repository.Colorschemes = colorschemes
		updateRepositoryAfterGenerate(repository)
	}

	cleanUp()

	return map[string]interface{}{
		"repositoryCount":        len(repositories),
		"repositoryErrorCount":   repositoryErrorCount,
		"repositoryErrorSamples": repositoryErrorSamples,
	}
}

func updateRepositoryAfterGenerate(repository repoHelper.Repository) {
	log.Printf("Generated %d colorschemes", len(repository.Colorschemes))
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
	colorschemeListFilePath = fmt.Sprintf("%s/repository_colorschemes.json", tmpDirectoryPath)
	colorschemeBatchFilePath = fmt.Sprintf("%s/colorscheme_batch.json", tmpDirectoryPath)

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

// Sets up the runtime configuration common to all colorschemes
func setupRuntime() {
	log.Print("Setting up runtime config")

	baseVimrcContent, err := file.GetLocalFileContent(fmt.Sprintf("%s/init.lua", vimFilesPath))
	if err != nil {
		log.Panic(err)
	}

	myVimrc := fmt.Sprintf("vim.env.MYVIMRC=\"%s\"\n", vimrcPath)

	runtimepath := fmt.Sprintf("vim.opt.runtimepath:append(\"%s\")\n", tmpDirectoryPath)
	packpath := fmt.Sprintf("vim.opt.packpath:append(\"%s\")\n", tmpDirectoryPath)

	vimrcContent := fmt.Sprintf("%s\n%s\n%s\n%s\n", baseVimrcContent, myVimrc, runtimepath, packpath)

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

// captureDefaultColorschemes records the colorschemes present before any repository is installed.
func captureDefaultColorschemes() {
	log.Print("Capturing default colorschemes")

	names, err := captureColorschemeNames(defaultColorschemeFilePath)
	if err != nil {
		log.Panic(err)
	}

	defaultColorschemes = make(map[string]bool, len(names))
	for _, name := range names {
		defaultColorschemes[name] = true
	}

	log.Printf("Captured %d default colorschemes", len(defaultColorschemes))
}

// captureColorschemeNames lists every colorscheme currently on the runtime path.
func captureColorschemeNames(outputPath string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), previewGenerationTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "nvim", "-u", vimrcPath, "--headless",
		"-c", captureColorschemeNamesCommand(outputPath))

	log.Printf("Running %s (timeout: %s)", cmd, previewGenerationTimeout)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return nil, wrapCommandError(ctx, "listing colorschemes", err)
	}

	content, err := file.GetLocalFileContent(outputPath)
	if err != nil {
		return nil, err
	}

	var names []string
	err = json.Unmarshal([]byte(content), &names)
	if err != nil {
		return nil, err
	}

	return names, nil
}

func captureColorschemeNamesCommand(outputPath string) string {
	return fmt.Sprintf(
		"lua local ok, err = pcall(require('extractor').colorschemes, { output_path = '%s' }) if not ok then io.stderr:write(tostring(err)) vim.cmd('cquit 1') else vim.cmd('qa!') end",
		outputPath,
	)
}

// Installs a plugin/colorscheme on the runtime configuration from a Github URL
func installPlugin(gitRepositoryURL string, path string) error {
	log.Printf("Installing %s", path)

	target := fmt.Sprintf("%s/%s", packDirectoryPath, path)

	ctx, cancel := context.WithTimeout(context.Background(), previewGenerationTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "clone", gitRepositoryURL, target)

	log.Printf("Running %s (timeout: %s)", cmd, previewGenerationTimeout)

	err := cmd.Run()
	if err != nil {
		return wrapCommandError(ctx, "installing "+path, err)
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
func getColorschemeColorData() (map[string]repoHelper.ColorschemeData, error) {
	names, err := captureColorschemeNames(colorschemeListFilePath)
	if err != nil {
		return nil, err
	}

	targets := repositoryColorschemes(names, defaultColorschemes)
	if len(targets) == 0 {
		log.Print("No colorscheme to extract")
		return map[string]repoHelper.ColorschemeData{}, nil
	}

	batches := chunkColorschemes(targets, colorschemeBatchSize)
	log.Printf("Extracting %d colorschemes in %d batches", len(targets), len(batches))

	data := map[string]repoHelper.ColorschemeData{}

	for index, batch := range batches {
		log.Printf("Extracting batch %d/%d (%d colorschemes)", index+1, len(batches), len(batch))

		batchData, err := extractColorData(batch)
		if err != nil {
			return nil, err
		}

		mergeColorData(data, batchData)
	}

	return data, nil
}

func repositoryColorschemes(names []string, defaults map[string]bool) []string {
	seen := make(map[string]bool, len(names))
	targets := []string{}

	for _, name := range names {
		if name == "" || defaults[name] || isDefaultColorscheme(name) || seen[name] {
			continue
		}

		seen[name] = true
		targets = append(targets, name)
	}

	// Stable order keeps batches identical from one run to the next.
	sort.Strings(targets)

	return targets
}

func chunkColorschemes(names []string, size int) [][]string {
	if size < 1 {
		size = 1
	}

	batches := [][]string{}

	for start := 0; start < len(names); start += size {
		end := start + size
		if end > len(names) {
			end = len(names)
		}

		batches = append(batches, names[start:end])
	}

	return batches
}

func mergeColorData(target map[string]repoHelper.ColorschemeData, source map[string]repoHelper.ColorschemeData) {
	for name, data := range source {
		if _, exists := target[name]; exists {
			continue
		}

		target[name] = data
	}
}

func extractColorData(colorschemes []string) (map[string]repoHelper.ColorschemeData, error) {
	batch, err := json.Marshal(colorschemes)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(colorschemeBatchFilePath, batch, os.FileMode(0600)); err != nil {
		return nil, err
	}

	if err := executePreviewGenerator(); err != nil {
		log.Printf("Error executing nvim: %s", err)
		return nil, err
	}

	colorSchemeOutput, err := file.GetLocalFileContent(colorDataFilePath)
	if err != nil {
		log.Printf("Error getting local file content from \"%s\": %s", colorDataFilePath, err)
		return nil, err
	}

	data, err := parseColorData([]byte(colorSchemeOutput))
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

func parseColorData(content []byte) (map[string]repoHelper.ColorschemeData, error) {
	// Empty content means a broken extractor run; failing here beats wiping stored colorschemes.
	trimmed := bytes.TrimSpace(content)

	var data map[string]repoHelper.ColorschemeData
	if err := json.Unmarshal(trimmed, &data); err != nil {
		return nil, err
	}

	if data == nil {
		data = map[string]repoHelper.ColorschemeData{}
	}

	return data, nil
}

// Starts a runtime instance and auto commands to configure and start vcspg on load
func executePreviewGenerator() error {
	args := []string{"-u", vimrcPath}

	if !debugMode {
		args = append(args, "--headless")
	}

	args = append(args, "./vim/code_sample.vim", "-c", extractCommand())

	if !debugMode {
		args = append(args, "-c", "qa!")
	}

	ctx, cancel := context.WithTimeout(context.Background(), previewGenerationTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "nvim", args...)

	log.Printf("Running %s (timeout: %s)", cmd, previewGenerationTimeout)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return wrapCommandError(ctx, "preview generation", err)
	}

	return nil
}

// The batch is read from a file since a colorscheme name may contain quotes.
// nvim exits 0 even when a -c command throws, hence the explicit cquit.
func extractCommand() string {
	return fmt.Sprintf(
		"lua local ok, err = pcall(require('extractor').extract, { colorschemes = vim.json.decode(table.concat(vim.fn.readfile('%s'), '')), output_path = '%s' }) if not ok then io.stderr:write(tostring(err)) vim.cmd('cquit 1') end",
		colorschemeBatchFilePath,
		colorDataFilePath,
	)
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
		Colorschemes: repository.Colorschemes,
	}
}

// wrapCommandError returns an error message that distinguishes timeout from other failures.
func wrapCommandError(ctx context.Context, action string, err error) error {
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("%s timed out after %s", action, previewGenerationTimeout)
	}
	return fmt.Errorf("%s failed: %w", action, err)
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

func appendRepositoryErrorSample(samples []string, repository repoHelper.Repository, err error) []string {
	if len(samples) >= 5 {
		return samples
	}

	return append(samples, fmt.Sprintf("%s/%s: %s", repository.Owner.Name, repository.Name, err))
}
