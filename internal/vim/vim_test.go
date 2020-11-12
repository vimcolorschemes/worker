package vim

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/vimcolorschemes/worker/internal/test"
)

func TestGetVimColorSchemeNames(t *testing.T) {
	t.Run("should return vim color scheme names of multiple valid files", func(t *testing.T) {
		fileContent1 := `
			hi clear
			syntax reset
			let g:colors_name = "test"
		`
		fileContent2 := `
			hi clear
			syntax reset
			let g:colors_name = "hello"
		`

		server1 := test.MockServer(fileContent1, http.StatusOK)
		defer server1.Close()

		server2 := test.MockServer(fileContent2, http.StatusOK)
		defer server2.Close()

		names := GetVimColorSchemeNames([]string{server1.URL, server2.URL})

		expectedNames := []string{"test", "hello"}

		if !reflect.DeepEqual(names, expectedNames) {
			t.Errorf("Incorrect result for GetVimColorSchemeNames, got: %s, want: %s", names, expectedNames)
		}
	})

	t.Run("should handle duplicate vim color scheme names", func(t *testing.T) {
		fileContent1 := `
			hi clear
			syntax reset
			let g:colors_name = "hello"
		`
		fileContent2 := `
			hi clear
			syntax reset
			let g:colors_name = "hello"
		`

		server1 := test.MockServer(fileContent1, http.StatusOK)
		defer server1.Close()

		server2 := test.MockServer(fileContent2, http.StatusOK)
		defer server2.Close()

		names := GetVimColorSchemeNames([]string{server1.URL, server2.URL})

		expectedNames := []string{"hello"}

		if !reflect.DeepEqual(names, expectedNames) {
			t.Errorf("Incorrect result for GetVimColorSchemeNames, got: %s, want: %s", names, expectedNames)
		}
	})

	t.Run("should return empty array on invalid vim color scheme files", func(t *testing.T) {
		fileContent1 := `
			hi clear
			syntax reset
		`
		fileContent2 := `
			hi clear
			syntax reset
		`

		server1 := test.MockServer(fileContent1, http.StatusOK)
		defer server1.Close()

		server2 := test.MockServer(fileContent2, http.StatusOK)
		defer server2.Close()

		names := GetVimColorSchemeNames([]string{server1.URL, server2.URL})

		expectedNames := []string{}

		if !reflect.DeepEqual(names, expectedNames) {
			t.Errorf("Incorrect result for GetVimColorSchemeNames, got: %s, want: %s", names, expectedNames)
		}
	})

	t.Run("should ignore invalid file URLs", func(t *testing.T) {
		fileContent := `
			hi clear
			syntax reset
			let g:colors_name = "test"
		`

		server := test.MockServer(fileContent, http.StatusOK)
		defer server.Close()

		names := GetVimColorSchemeNames([]string{server.URL, "wrong url"})

		expectedNames := []string{"test"}

		if !reflect.DeepEqual(names, expectedNames) {
			t.Errorf("Incorrect result for GetVimColorSchemeNames, got: %s, want: %s", names, expectedNames)
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
		syntax reset`, name: "hello_world"},
	{fileContent: `
		hi clear
		let g:color_name="hello (world)"
		syntax reset`, name: "hello (world)"},
	{fileContent: `
		hi clear
		let colors_name = "abcd1234"
		syntax reset`, name: "abcd1234"},
	{fileContent: `
		hi clear
		let color_name = "TEST"
		syntax reset`, name: "TEST"},
	{fileContent: `
		hi clear
		let colors_name="TEst"
		syntax reset`, name: "TEst"},
	{fileContent: `
		hi clear
		let color_name="test"
		syntax reset`, name: "test"},
	{fileContent: `
		hi clear
		let g:colors_name = 'test'
		syntax reset`, name: "test"},
	{fileContent: `
		hi clear
		let g:color_name = 'test'
		syntax reset`, name: "test"},
	{fileContent: `
		hi clear
		let g:colors_name='test'
		syntax reset`, name: "test"},
	{fileContent: `
		hi clear
		let g:color_name='test'
		syntax reset`, name: "test"},
	{fileContent: `
		hi clear
		let colors_name = 'test'
		syntax reset`, name: "test"},
	{fileContent: `
		hi clear
		let color_name = 'test'
		syntax reset`, name: "test"},
	{fileContent: `
		hi clear
		let colors_name='test'
		syntax reset`, name: "test"},
	{fileContent: `
		hi clear
		let color_name='test'
		syntax reset`, name: "test"},
}

func TestGetVimColorSchemeName(t *testing.T) {
	t.Run("should return the vim color scheme name if the file is valid", func(t *testing.T) {
		for _, item := range validTests {
			name, err := GetVimColorSchemeName(&item.fileContent)
			if err != nil {
				t.Error("Incorrect result for GetVimColorSchemeName, got error")
			}
			if name != item.name {
				t.Errorf("Incorrect result for GetVimColorSchemeName, got: %s, want: %s", name, item.name)
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
			let colors_name = "{}"
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
			name, err := GetVimColorSchemeName(&fileContent)
			if err == nil {
				t.Error("Incorrect result for GetVimColorSchemeName, got no error")
			}
			if name != "" {
				t.Errorf("Incorrect result for GetVimColorSchemeName, got: %s, want: %s", name, "")
			}
		}
	})
}
