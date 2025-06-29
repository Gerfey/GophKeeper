package client

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/gerfey/gophkeeper/internal/models"
	"github.com/rivo/tview"
)

const (
	standardFieldWidth = 30
	shortFieldWidth    = 20
	longFieldWidth     = 50
	cvvFieldWidth      = 3
	textAreaHeight     = 10

	idColumn      = 0
	typeColumn    = 1
	nameColumn    = 2
	updatedColumn = 3

	dataTypeLoginPass = "Логин/Пароль" // #nosec G101
	dataTypeText      = "Текст"
	dataTypeCard      = "Карта"
	dataTypeFile      = "Файл"

	dialogWidth  = 60
	dialogHeight = 15

	passwordFieldWidth = 30

	syncIntervalSeconds = 15

	dialogFieldWidth = 50
	formPadding      = 3
	formWidth        = 60
	formSidePadding  = 20
)

type Config struct {
	ServerURL         string `json:"server_url"`
	Token             string `json:"token"`
	Username          string `json:"username"`
	HasMasterPassword bool   `json:"has_master_password"`
}

type TUI struct {
	app               *tview.Application
	pages             *tview.Pages
	client            *Client
	config            *Config
	configPath        string
	dataList          []models.DataResponse
	setViewData       func(data models.DataResponse)
	lastFileDialogDir string
	updateTable       func()
	syncTimer         *time.Timer
	dataTable         *tview.Table
}

func NewTUI(configPath string) (*TUI, error) {
	tui := &TUI{
		app:        tview.NewApplication(),
		pages:      tview.NewPages(),
		configPath: configPath,
		config: &Config{
			ServerURL: "https://localhost:8080",
		},
	}

	if err := tui.loadConfig(); err == nil {
		tui.client = NewClient(tui.config.ServerURL, true)
		tui.client.SetAuthToken(tui.config.Token)
	} else {
		tui.client = NewClient(tui.config.ServerURL, true)
	}

	return tui, nil
}

func (t *TUI) loadConfig() error {
	if _, err := os.Stat(t.configPath); os.IsNotExist(err) {
		return err
	}

	data, err := os.ReadFile(t.configPath)
	if err != nil {
		return err
	}

	if unmarshalErr := json.Unmarshal(data, t.config); unmarshalErr != nil {
		return unmarshalErr
	}

	return nil
}

func (t *TUI) saveConfig() error {
	dir := filepath.Dir(t.configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.Marshal(t.config)
	if err != nil {
		return err
	}

	return os.WriteFile(t.configPath, data, 0600)
}

func (t *TUI) Run() error {
	t.initPages()

	if t.config.Token == "" {
		t.pages.SwitchToPage("login")
	} else {
		t.loadData()
		t.startAutoSync()
		t.pages.SwitchToPage("main")
	}

	return t.app.SetRoot(t.pages, true).EnableMouse(true).Run()
}

func (t *TUI) startAutoSync() {
	t.syncTimer = time.AfterFunc(syncIntervalSeconds*time.Second, func() {
		t.app.QueueUpdateDraw(func() {
			t.loadData()
		})
		t.startAutoSync()
	})
}

func (t *TUI) stopAutoSync() {
	if t.syncTimer != nil {
		t.syncTimer.Stop()
		t.syncTimer = nil
	}
}

func (t *TUI) initPages() {
	t.pages.AddPage("login", t.createLoginPage(), true, true)
	t.pages.AddPage("register", t.createRegisterPage(), true, false)
	t.pages.AddPage("main", t.createMainPage(), true, false)
	t.pages.AddPage("add", t.createAddPage(), true, false)
	t.pages.AddPage("view", t.createViewPage(), true, false)
}

func (t *TUI) getDataTypeLabel(dataType models.DataType) string {
	switch dataType {
	case models.LoginPassword:
		return dataTypeLoginPass
	case models.TextData:
		return dataTypeText
	case models.CardData:
		return dataTypeCard
	case models.BinaryData:
		return dataTypeFile
	default:
		return string(dataType)
	}
}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}
