package docker

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	dockerfileName = "Dockerfile"
	dibPlaceholder = "DIB_MANAGED_VERSION"
)

var (
	rxFrom  *regexp.Regexp
	rxLabel *regexp.Regexp
)

func init() {
	rxFrom = regexp.MustCompile(`^FROM (\S+):\S+( as \S+)?$`)
	rxLabel = regexp.MustCompile(`^LABEL (\S+)="(\S+)"$`)
}

// Dockerfile holds the information from a Dockerfile.
type Dockerfile struct {
	ContextPath string
	Filename    string
	From        []string
	Labels      map[string]string
}

func (d *Dockerfile) addFrom(from string) {
	d.From = append(d.From, from)
}

func (d *Dockerfile) addLabel(name string, value string) {
	d.Labels[name] = value
}

// IsDockerfile checks whether a file is a Dockerfile.
func IsDockerfile(filename string) bool {
	return strings.HasSuffix(filename, dockerfileName)
}

// ParseDockerfile parses an actual Dockerfile, and creates an instance of a Dockerfile struct.
func ParseDockerfile(filename string) (*Dockerfile, error) {
	logrus.Debugf("Parsing dockerfile \"%s\"", filename)
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var d Dockerfile
	d.ContextPath = path.Dir(filename)
	d.Filename = path.Base(filename)
	d.Labels = map[string]string{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		txt := scanner.Text()

		switch {
		case rxFrom.MatchString(txt):
			d.addFrom(rxFrom.FindStringSubmatch(txt)[1])
		case rxLabel.MatchString(txt):
			result := rxLabel.FindStringSubmatch(txt)
			d.addLabel(result[1], result[2])
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	logrus.Debugf("Successfully parsed dockerfile. From=%v, Labels=%v", d.From, d.Labels)

	return &d, nil
}

// ReplaceFromTag replaces the tag placeholder in a Dockerfile with an actual tag.
func ReplaceFromTag(d Dockerfile, tag string) error {
	err := replace(path.Join(d.ContextPath, d.Filename), dibPlaceholder, tag)
	if err != nil {
		return fmt.Errorf("cannot replace placeholder tag: %w", err)
	}

	return nil
}

// ResetFromTag replaces the tag in a Dockerfile back to the original placeholder.
func ResetFromTag(d Dockerfile, tag string) error {
	err := replace(path.Join(d.ContextPath, d.Filename), tag, dibPlaceholder)
	if err != nil {
		return fmt.Errorf("cannot reset tag: %w", err)
	}

	return nil
}

func replace(path string, previous string, next string) error {
	read, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	newContents := strings.ReplaceAll(string(read), previous, next)
	if err = ioutil.WriteFile(path, []byte(newContents), 0); err != nil {
		return err
	}
	return nil
}
