package plaintext

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/v2fly/geoip/lib"
)

const (
	typeTextOut = "text"
	descTextOut = "Convert data to plaintext CIDR format"
)

var (
	defaultOutputDir = filepath.Join("./", "output", "text")
)

func init() {
	lib.RegisterOutputConfigCreator(typeTextOut, func(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
		return newTextOut(action, data)
	})
	lib.RegisterOutputConverter(typeTextOut, &textOut{
		Description: descTextOut,
	})
}

func newTextOut(action lib.Action, data json.RawMessage) (lib.OutputConverter, error) {
	var tmp struct {
		OutputDir  string     `json:"outputDir"`
		Want       []string   `json:"wantedList"`
		OnlyIPType lib.IPType `json:"onlyIPType"`

		AddPrefixInLine string `json:"addPrefixInLine"`
		AddSuffixInLine string `json:"addSuffixInLine"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &tmp); err != nil {
			return nil, err
		}
	}

	if tmp.OutputDir == "" {
		tmp.OutputDir = defaultOutputDir
	}

	// Filter want list
	wantList := make([]string, 0, len(tmp.Want))
	for _, want := range tmp.Want {
		if want = strings.ToUpper(strings.TrimSpace(want)); want != "" {
			wantList = append(wantList, want)
		}
	}

	return &textOut{
		Type:        typeTextOut,
		Action:      action,
		Description: descTextOut,
		OutputDir:   tmp.OutputDir,
		Want:        wantList,
		OnlyIPType:  tmp.OnlyIPType,

		AddPrefixInLine: tmp.AddPrefixInLine,
		AddSuffixInLine: tmp.AddSuffixInLine,
	}, nil
}

type textOut struct {
	Type        string
	Action      lib.Action
	Description string
	OutputDir   string
	Want        []string
	OnlyIPType  lib.IPType

	AddPrefixInLine string
	AddSuffixInLine string
}

func (t *textOut) GetType() string {
	return t.Type
}

func (t *textOut) GetAction() lib.Action {
	return t.Action
}

func (t *textOut) GetDescription() string {
	return t.Description
}

func (t *textOut) Output(container lib.Container) error {
	switch len(t.Want) {
	case 0:
		list := make([]string, 0, 300)
		for entry := range container.Loop() {
			list = append(list, entry.GetName())
		}

		// Sort the list
		slices.Sort(list)

		for _, name := range list {
			entry, found := container.GetEntry(name)
			if !found {
				log.Printf("❌ entry %s not found", name)
				continue
			}
			cidrList, err := t.marshalText(entry)
			if err != nil {
				return err
			}
			filename := strings.ToLower(entry.GetName()) + ".txt"
			if err := t.writeFile(filename, cidrList); err != nil {
				return err
			}
		}

	default:
		// Sort the list
		slices.Sort(t.Want)

		for _, name := range t.Want {
			entry, found := container.GetEntry(name)
			if !found {
				log.Printf("❌ entry %s not found", name)
				continue
			}
			cidrList, err := t.marshalText(entry)
			if err != nil {
				return err
			}
			filename := strings.ToLower(entry.GetName()) + ".txt"
			if err := t.writeFile(filename, cidrList); err != nil {
				return err
			}
		}
	}

	return nil
}

func (t *textOut) marshalText(entry *lib.Entry) ([]string, error) {
	var entryCidr []string
	var err error
	switch t.OnlyIPType {
	case lib.IPv4:
		entryCidr, err = entry.MarshalText(lib.IgnoreIPv6)
		if err != nil {
			return nil, err
		}
	case lib.IPv6:
		entryCidr, err = entry.MarshalText(lib.IgnoreIPv4)
		if err != nil {
			return nil, err
		}
	default:
		entryCidr, err = entry.MarshalText()
		if err != nil {
			return nil, err
		}
	}

	return entryCidr, nil
}

func (t *textOut) writeFile(filename string, cidrList []string) error {
	var buf bytes.Buffer
	for _, cidr := range cidrList {
		if t.AddPrefixInLine != "" {
			buf.WriteString(t.AddPrefixInLine)
		}
		buf.WriteString(cidr)
		if t.AddSuffixInLine != "" {
			buf.WriteString(t.AddSuffixInLine)
		}
		buf.WriteString("\n")
	}
	cidrBytes := buf.Bytes()

	if err := os.MkdirAll(t.OutputDir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(t.OutputDir, filename), cidrBytes, 0644); err != nil {
		return err
	}

	log.Printf("✅ [%s] %s --> %s", t.Type, filename, t.OutputDir)

	return nil
}
