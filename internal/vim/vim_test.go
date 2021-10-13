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
			{Name: "test", FileURL: server1.URL, UsesXtermColors: true},
			{Name: "hello", FileURL: server2.URL, UsesXtermColors: true},
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

		expectedVimColorSchemes := []repository.VimColorScheme{{Name: "hello", FileURL: server.URL, UsesXtermColors: true}}

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

		expectedVimColorSchemes := []repository.VimColorScheme{{Name: "test", FileURL: server.URL, UsesXtermColors: true}}

		if !reflect.DeepEqual(colorSchemes, expectedVimColorSchemes) {
			var names []string
			for _, colorScheme := range colorSchemes {
				names = append(names, colorScheme.Name)
			}
			t.Errorf("Incorrect result for GetVimColorSchemes, got: %s, want: %s", names, []string{"test"})
		}
	})
}

var validTests = []struct {
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

func TestGetVimColorSchemeName(t *testing.T) {
	t.Run("should return the vim color scheme name if the file is valid", func(t *testing.T) {
		for _, item := range validTests {
			name, err := getVimColorSchemeName(&item.fileContent)
			if err != nil {
				t.Error("Incorrect result for getVimColorSchemeName, got error")
			}
			if name != item.name {
				t.Errorf("Incorrect result for getVimColorSchemeName, got: %s, want: %s", name, item.name)
			}
		}
	})

	invalid := []string{
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
		for _, fileContent := range invalid {
			name, err := getVimColorSchemeName(&fileContent)
			if err == nil {
				t.Error("Incorrect result for getVimColorSchemeName, got no error")
			}
			if name != "" {
				t.Errorf("Incorrect result for getVimColorSchemeName, got: %s, want: %s", name, "")
			}
		}
	})
}

func TestIsVimColorScheme(t *testing.T) {
	valid := []string{
		`
			exe "hi! Normal"        .s:fg_foreground  .s:bg_normal      .s:fmt_none
		`,
		`
			hi Normal ctermbg=254 ctermfg=237 guibg=#e8e9ec guifg=#33374c
		`,
		`
			gitrebaseSummary = 'Normal'
		`,
	}
	t.Run("should return true if the file has necessary content", func(t *testing.T) {
		for _, fileContent := range valid {
			if !isVimColorScheme(&fileContent) {
				t.Errorf("Incorrect result for isVimColorScheme, got false for: %s", fileContent)
			}
		}
	})

	invalid := []string{
		`
			normal ctermbg=254 ctermfg=237 guibg=#e8e9ec guifg=#33374c
		`,
	}
	t.Run("should return false if the file does not have necessary content", func(t *testing.T) {
		for _, fileContent := range invalid {
			if isVimColorScheme(&fileContent) {
				t.Errorf("Incorrect result for isVimColorScheme, got true for: %s", fileContent)
			}
		}
	})
}

func TestGetSupportsTermGuiColors(t *testing.T) {
	t.Run("should return true if 5 or more hex codes (with pound) are found", func(t *testing.T) {
		content := `
			hi clear
			let g:colors_name = "test
			syntax reset
			normal ctermbg=254 ctermfg=237 guibg=#e8e9ec guifg=#33374c
			normal ctermbg=254 ctermfg=237 guibg=#e8e9ec guifg=#33374c
			normal ctermbg=254 ctermfg=237 guibg=#e8e9ec guifg=#33374c
		`

		result := getSupportsTermGuiColors(&content)

		if result == false {
			t.Errorf("Incorrect result for getSupportsTermGuiColors, got %v, want: %v", result, true)
		}
	})

	t.Run("should return true if 5 or more hex codes (without pound) are found", func(t *testing.T) {
		content := `
			hi clear
			let g:colors_name = "test
			syntax reset

			let s:gui00 = "263238"
			let s:gui01 = "37474F"
			let s:gui02 = "546E7A"
			let s:gui03 = "5C7E8C"
			let s:gui04 = "80CBC4"

			call <sid>hi("Search",        s:gui03, s:gui0A, s:cterm03, s:cterm0A,  "")
			call <sid>hi("SpecialKey",    s:gui03, "", s:cterm03, "", "")
			call <sid>hi("TooLong",       s:gui08, "", s:cterm08, "", "")
		`

		result := getSupportsTermGuiColors(&content)

		if result == false {
			t.Errorf("Incorrect result for getSupportsTermGuiColors, got %v, want: %v", result, true)
		}
	})

	t.Run("should return false if less than 5 hex codes are found", func(t *testing.T) {
		content := `
			hi clear
			let g:colors_name = "test
			syntax reset
			normal ctermbg=254 ctermfg=237 guibg=#e8e9ec guifg=#33374c
			normal ctermbg=254 ctermfg=237 guibg=#e8e9ec guifg=#33374c
		`

		result := getSupportsTermGuiColors(&content)

		if result == true {
			t.Errorf("Incorrect result for getSupportsTermGuiColors, got %v, want: %v", result, false)
		}
	})

	t.Run("should return false if no hex codes are found", func(t *testing.T) {
		content := `
			hi clear
			let g:colors_name = "test
			syntax reset
			normal ctermbg=254 ctermfg=237
			normal ctermbg=254 ctermfg=237
		`

		result := getSupportsTermGuiColors(&content)

		if result == true {
			t.Errorf("Incorrect result for getSupportsTermGuiColors, got %v, want: %v", result, false)
		}
	})

	t.Run("should return false if gui_running check is present", func(t *testing.T) {
		content := `
			hi clear
			let g:colors_name = "test
			syntax reset
			normal ctermbg=254 ctermfg=237
			normal ctermbg=254 ctermfg=237

			let s:gui00 = "#263238"
			let s:gui01 = "#37474F"
			let s:gui02 = "#546E7A"
			let s:gui03 = "#5C7E8C"
			let s:gui04 = "#80CBC4"

			if has('gui_running')
				echo "yes"
			else
				echo "no"
			endif
		`

		result := getSupportsTermGuiColors(&content)

		if result == true {
			t.Errorf("Incorrect result for getSupportsTermGuiColors, got %v, want: %v", result, false)
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
