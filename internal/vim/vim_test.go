package vim

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/vimcolorschemes/worker/internal/repository"
	"github.com/vimcolorschemes/worker/internal/test"
)

func TestGetVimColorSchemes(t *testing.T) {
	t.Run("should return vim color scheme file URLs of multiple valid files", func(t *testing.T) {
		fileContent1 := `
			hi clear
			syntax reset
			let g:colors_name = "test"
			hi Normal cterm=97
		`
		fileContent2 := `
			hi clear
			syntax reset
			let g:colors_name = "hello"
			hi Normal cterm=97
		`

		server1 := test.MockServer(fileContent1, http.StatusOK)
		defer server1.Close()

		server2 := test.MockServer(fileContent2, http.StatusOK)
		defer server2.Close()

		colorSchemes, err := GetVimColorSchemes([]string{server1.URL, server2.URL})

		if err != nil {
			t.Errorf("Incorrect result for GetVimColorSchemes, got error: %s", err)
		}

		expectedVimColorSchemes := []repository.VimColorScheme{
			{Name: "test", FileURL: server1.URL},
			{Name: "hello", FileURL: server2.URL},
		}

		if !reflect.DeepEqual(colorSchemes, expectedVimColorSchemes) {
			var names []string
			for _, colorScheme := range colorSchemes {
				names = append(names, colorScheme.Name)
			}
			t.Errorf("Incorrect result for GetVimColorSchemes, got: %s, want: %s", names, []string{"test", "hello"})
		}
	})

	t.Run("should handle duplicate vim color scheme file URLs", func(t *testing.T) {
		fileContent := `
			hi clear
			syntax reset
			let g:colors_name = "hello"
			hi Normal cterm=97
		`

		server := test.MockServer(fileContent, http.StatusOK)
		defer server.Close()

		colorSchemes, err := GetVimColorSchemes([]string{server.URL, server.URL})

		if err != nil {
			t.Error("Incorrect result for GetVimColorSchemes, got error")
		}

		expectedVimColorSchemes := []repository.VimColorScheme{{Name: "hello", FileURL: server.URL}}

		if !reflect.DeepEqual(colorSchemes, expectedVimColorSchemes) {
			var names []string
			for _, colorScheme := range colorSchemes {
				names = append(names, colorScheme.Name)
			}
			t.Errorf("Incorrect result for GetVimColorSchemes, got: %s, want: %s", names, []string{"hello"})
		}
	})

	t.Run("should return empty array and error on invalid vim color scheme files", func(t *testing.T) {
		fileContent1 := `
			hi clear
			syntax reset
			let g:color='hello'
		`
		fileContent2 := `
			hi clear
			syntax reset
			hi Normal cterm=97
		`
		fileContent3 := `
			hi clear
			syntax reset
			let g:color='test'
			hi Normal cterm=97
		`

		server1 := test.MockServer(fileContent1, http.StatusOK)
		defer server1.Close()

		server2 := test.MockServer(fileContent2, http.StatusOK)
		defer server2.Close()

		server3 := test.MockServer(fileContent3, http.StatusOK)
		defer server3.Close()

		colorSchemes, err := GetVimColorSchemes([]string{server1.URL, server2.URL, server3.URL})

		expectedVimColorSchemes := []repository.VimColorScheme{}

		if err == nil {
			t.Error("Incorrect result for GetVimColorSchemes, got no error")
		}

		if !reflect.DeepEqual(colorSchemes, expectedVimColorSchemes) {
			var names []string
			for _, colorScheme := range colorSchemes {
				names = append(names, colorScheme.Name)
			}
			t.Errorf("Incorrect result for GetVimColorSchemes, got: %s, want: %s", names, []string{})
		}
	})

	t.Run("should ignore invalid file URLs", func(t *testing.T) {
		fileContent := `
			hi clear
			syntax reset
			let g:colors_name = "test"
			hi Normal cterm=97
		`

		server := test.MockServer(fileContent, http.StatusOK)
		defer server.Close()

		colorSchemes, err := GetVimColorSchemes([]string{server.URL, "wrong url"})

		if err != nil {
			t.Error("Incorrect result for GetVimColorSchemes, got error")
		}

		expectedVimColorSchemes := []repository.VimColorScheme{{Name: "test", FileURL: server.URL}}

		if !reflect.DeepEqual(colorSchemes, expectedVimColorSchemes) {
			var names []string
			for _, colorScheme := range colorSchemes {
				names = append(names, colorScheme.Name)
			}
			t.Errorf("Incorrect result for GetVimColorSchemes, got: %s, want: %s", names, []string{"test"})
		}
	})
}

func TestGetColorSchemeName(t *testing.T) {
	var validVimTests = []struct {
		fileContent string
		name        string
	}{
		{fileContent: `
		hi clear
		let g:colors_name = "test"
		syntax reset`, name: "test"},
		{fileContent: `
		hi clear
		let g:color_name = "hello-world"
		syntax reset`, name: "hello-world"},
		{fileContent: `
		hi clear
		let g:colors_name="hello_world"
		syntex reset`, name: "hello_world"},
		{fileContent: `
		hi clear
		let g:color_name="hello (world)"
		syntex reset`, name: "helloworld"},
		{fileContent: `
		hi clear
		let colors_name = "abcd1234"
		syntex reset`, name: "abcd1234"},
		{fileContent: `
		hi clear
		let color_name = "TEST"
		syntex reset`, name: "test"},
		{fileContent: `
		hi clear
		let colors_name="TEst"
		syntex reset`, name: "test"},
		{fileContent: `
		hi clear
		let color_name="test"
		syntex reset`, name: "test"},
		{fileContent: `
		hi clear
		let g:colors_name = 'test'
		syntex reset`, name: "test"},
		{fileContent: `
		hi clear
		let g:color_name = 'test'
		syntex reset`, name: "test"},
		{fileContent: `
		hi clear
		let g:colors_name='test'
		syntex reset`, name: "test"},
		{fileContent: `
		hi clear
		let g:color_name='test'
		syntex reset`, name: "test"},
		{fileContent: `
		hi clear
		let colors_name = 'test'
		syntex reset`, name: "test"},
		{fileContent: `
		hi clear
		let color_name = 'test'
		syntex reset`, name: "test"},
		{fileContent: `
		hi clear
		let colors_name='test'
		syntex reset`, name: "test"},
		{fileContent: `
		hi clear
		let color_name='test'
		syntex reset`, name: "test"},
		{fileContent: `
		hi clear
		vim.g.colors_name = 'highlite'
		syntex reset`, name: "highlite"},
	}
	t.Run("should return the vim color scheme name if the file is valid", func(t *testing.T) {
		for _, item := range validVimTests {
			name, isLua, err := getColorSchemeName(&item.fileContent)
			if err != nil {
				t.Error("Incorrect result for getColorSchemeName, got error")
			}
			if name != item.name {
				t.Errorf("Incorrect result for getColorSchemeName, got: %s, want: %s", name, item.name)
			}
			if isLua {
				t.Error("Incorrect result for getColorSchemeName, got: isLua=true, want: false")
			}
		}
	})

	invalidVimTests := []string{
		`
			hi clear
			let g:colors_name = "test
			syntax reset
		`,
		`
			hi clear
			let g:color_names = "test"
			syntax reset
		`,
		`
			hi clear
			g:colors_name="test"
			syntax reset
		`,
		`
			hi clear
			let g:color_name="'test'"
			syntax reset
		`,
		`
			hi clear
			let colors_name = "{test}"
			syntax reset
		`,
		`
			hi clear
			let color_name = expand("test")
			syntax reset
		`,
	}
	t.Run("should return an error if the file is invalid", func(t *testing.T) {
		for _, fileContent := range invalidVimTests {
			name, isLua, err := getColorSchemeName(&fileContent)
			if err == nil {
				t.Error("Incorrect result for getColorSchemeName, got no error")
			}
			if name != "" {
				t.Errorf("Incorrect result for getColorSchemeName, got: %s, want: %s", name, "")
			}
			if isLua {
				t.Error("Incorrect result for getColorSchemeName, got: isLua=true, want: false")
			}
		}
	})

	var validLuaTests = []struct {
		fileContent string
		name        string
	}{
		{fileContent: `
    lua << EOF
    -- Useful for me when I am trying to debug and reload my changes
    if vim.g.nightfox_debug == true then
      package.loaded['nightfox'] = nil
      package.loaded['nightfox.colors'] = nil
      package.loaded["nightfox.colors.dawnfox"] = nil
      package.loaded['nightfox.theme'] = nil
      package.loaded['nightfox.util'] = nil
    end

    local nightfox = require('nightfox')
    nightfox.setup({fox = "dawnfox"})
    nightfox._colorscheme_load()
    EOF`, name: "nightfox"},
		{fileContent: `
		" Theme: zephyr
    " Author: Glepnir
    " License: MIT
    " Source: http://github.com/glepnir/zephyr-nvim

    lua require('zephyr')`, name: "zephyr"},
	}
	t.Run("should return the lua color scheme name if the file is valid", func(t *testing.T) {
		for _, item := range validLuaTests {
			name, isLua, err := getColorSchemeName(&item.fileContent)
			if err != nil {
				t.Error("Incorrect result for getColorSchemeName, got error")
			}
			if name != item.name {
				t.Errorf("Incorrect result for getColorSchemeName, got: %s, want: %s", name, item.name)
			}
			if !isLua {
				t.Error("Incorrect result for getColorSchemeName, got: isLua=false, want: true")
			}
		}
	})

	invalidLuaTests := []string{
		`
      lua << EOF
      -- Useful for me when I am trying to debug and reload my changes
      if vim.g.nightfox_debug == true then
        package.loaded['nightfox'] = nil
        package.loaded['nightfox.colors'] = nil
        package.loaded["nightfox.colors.dawnfox"] = nil
        package.loaded['nightfox.theme'] = nil
        package.loaded['nightfox.util'] = nil
      end
		`,
		`
      require('nightfox')
		`,
	}
	t.Run("should return an error if the file is invalid", func(t *testing.T) {
		for _, fileContent := range invalidLuaTests {
			name, _, err := getColorSchemeName(&fileContent)
			if err == nil {
				t.Error("Incorrect result for getColorSchemeName, got no error")
			}
			if name != "" {
				t.Errorf("Incorrect result for getColorSchemeName, got: %s, want: %s", name, "")
			}
		}
	})
}

func TestContainsURL(t *testing.T) {
	t.Run("should return true if list contains URL", func(t *testing.T) {
		vimColorSchemes := []repository.VimColorScheme{{FileURL: "URL1"}, {FileURL: "URL2"}}

		result := containsURL(vimColorSchemes, "URL2")

		if !result {
			t.Errorf("Incorrect result for containsURL, got %v, want: %v", result, true)
		}
	})

	t.Run("should return false if list is empty", func(t *testing.T) {
		vimColorSchemes := []repository.VimColorScheme{}

		result := containsURL(vimColorSchemes, "URL1")

		if result {
			t.Errorf("Incorrect result for containsURL, got %v, want: %v", result, false)
		}
	})

	t.Run("should return false if list does not contain URL", func(t *testing.T) {
		vimColorSchemes := []repository.VimColorScheme{{FileURL: "URL1"}, {FileURL: "URL2"}}

		result := containsURL(vimColorSchemes, "URL3")

		if result {
			t.Errorf("Incorrect result for containsURL, got %v, want: %v", result, false)
		}
	})
}

func TestNormalizeVimColorSchemeColors(t *testing.T) {
	t.Run("should normalise status line if it has the same background and foreground", func(t *testing.T) {
		var colors = []repository.VimColorSchemeGroup{
			{
				Name:    "NormalBg",
				HexCode: "#000000",
			},
			{
				Name:    "NormalFg",
				HexCode: "#ffffff",
			},
			{
				Name:    "StatusLineBg",
				HexCode: "#cccccc",
			},
			{
				Name:    "StatusLineFg",
				HexCode: "#cccccc",
			},
		}

		normalizedColors := NormalizeVimColorSchemeColors(colors)

		var statusLineBg repository.VimColorSchemeGroup
		var statusLineFg repository.VimColorSchemeGroup

		for i := 0; i < len(normalizedColors); i++ {
			color := normalizedColors[i]
			if color.Name == "StatusLineBg" {
				statusLineBg = color
			}
			if color.Name == "StatusLineFg" {
				statusLineFg = color
			}
		}

		if statusLineBg.HexCode != "#000000" {
			t.Errorf("Incorrect result for NormalizeVimColorSchemeColors, got %s, want: %s", statusLineBg.HexCode, "#000000")
		}

		if statusLineFg.HexCode != "#ffffff" {
			t.Errorf("Incorrect result for NormalizeVimColorSchemeColors , got %s, want: %s", statusLineFg.HexCode, "#ffffff")
		}
	})
}
