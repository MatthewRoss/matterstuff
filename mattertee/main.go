package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/42wim/matterbridge/matterhook"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"time"
)

type config struct {
	Username, Channel, Title, Language string
	IconURL                            string `yaml:"icon_url"`
	MatterURL                          string `yaml:"matter_url"`
	PlainText                          bool   `yaml:"plain_text"`
	NoBuffer                           bool   `yaml:"no_buffer"`
	Extra                              bool
}

var cfg = config{
	Channel:   "",
	IconURL:   "",
	Language:  "",
	MatterURL: "",
	Title:     "",
	Username:  "mattertee",
	Extra:     false,
	NoBuffer:  false,
	PlainText: false,
}

func init() {
	// Read configuration from files
	read_configurations()

	// Now override configuration with command line parameters
	flag.StringVar(&cfg.Channel, "c", cfg.Channel, "Post input values to specified channel or user.")
	flag.StringVar(&cfg.IconURL, "i", cfg.IconURL, "This url is used as icon for posting.")
	flag.StringVar(&cfg.Language, "l", cfg.Language, "Specify the language used for syntax highlighting (ruby/python/...)")
	flag.StringVar(&cfg.MatterURL, "m", cfg.MatterURL, "Mattermost incoming webhooks URL.")
	flag.StringVar(&cfg.Title, "t", cfg.Title, "This title is added to posts. (not with -n)")
	flag.StringVar(&cfg.Username, "u", cfg.Username, "This username is used for posting.")
	flag.BoolVar(&cfg.Extra, "x", cfg.Extra, "Add extra info (user/hostname/timestamp).")
	flag.BoolVar(&cfg.NoBuffer, "n", cfg.NoBuffer, "Post input values without buffering.")
	flag.BoolVar(&cfg.PlainText, "p", cfg.PlainText, "Don't surround the post with triple backticks.")
	flag.Parse()
	fmt.Fprintf(os.Stderr, "Configuration: %v\n", cfg)
}

func read_configurations() {
	// config_files will list configuration files which will be read in order and can override
	// previous files
	config_files := []string{}

	if runtime.GOOS == "linux" {
		config_files = append(config_files, "/etc/mattertee.conf")
	}

	usr, err := user.Current()
	if err == nil {
		config_files = append(config_files, filepath.Join(usr.HomeDir, ".mattertee.conf"))
	}

	config_files = append(config_files, ".mattertee.conf")

	for _, file := range config_files {
		err := read_configuration(file)
		if err != nil {
			// something went wrong - report (but don't fail)
			fmt.Fprintf(os.Stderr, "An error has occurred while reading configuration file '%s': %s\n", file, err)
		}
	}
}

func read_configuration(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		// File doesn't exist, so skip it
		return nil
	}

	err = yaml.Unmarshal([]byte(data), &cfg)
	if err != nil {
		return err
	}

	return nil
}

func md(text string) string {
	return "```" + cfg.Language + "\n" + text + "```"
}

func extraInfo() string {
	u, _ := user.Current()
	hname, _ := os.Hostname()
	return "\n" + u.Username + "@" + hname + " (_" + time.Now().Format(time.RFC3339) + "_)\n"
}

func main() {
	url := os.Getenv("MM_HOOK")
	if cfg.MatterURL != "" {
		url = cfg.MatterURL
	}
	m := matterhook.New(url, matterhook.Config{DisableServer: true})
	msg := matterhook.OMessage{}
	msg.UserName = cfg.Username
	msg.Channel = cfg.Channel
	msg.IconURL = cfg.IconURL
	if cfg.NoBuffer {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
			msg.Text = md(line)
			if cfg.PlainText {
				msg.Text = line
			}
			m.Send(msg)
		}
	} else {
		buf := new(bytes.Buffer)
		io.Copy(buf, os.Stdin)
		text := buf.String()
		fmt.Print(text)
		msg.Text = md(text)
		if cfg.PlainText {
			msg.Text = text
		}
		if cfg.Extra {
			msg.Text += extraInfo()
		}
		if cfg.Title != "" {
			msg.Text = cfg.Title + "\n" + msg.Text
		}
		m.Send(msg)
	}
}
