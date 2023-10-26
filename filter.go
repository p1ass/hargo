package hargo

import (
	"bufio"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"os"
)

// Filter filters logs in .har file
func Filter(r *bufio.Reader, filename string) error {
	har, err := Decode(r)

	check(err)

	var filteredEntries []Entry
	for _, entry := range har.Log.Entries {

		if entry.Response.Content.MimeType == "application/json" {
			filteredEntries = append(filteredEntries, entry)
		}
	}

	har.Log.Entries = filteredEntries

	f, err := os.Create(filename)
	if err != nil {
		log.Error(err)
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")

	err = enc.Encode(har)
	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}
