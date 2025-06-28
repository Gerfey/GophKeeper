package client

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/gerfey/gophkeeper/internal/models"
	"github.com/rivo/tview"
)

var Version = "v1.0.0"

type Config struct {
	ServerURL string `json:"server_url"`
	Token     string `json:"token"`
	Username  string `json:"username"`
}

type TUI struct {
	app         *tview.Application
	pages       *tview.Pages
	client      *Client
	config      *Config
	configPath  string
	dataList    []models.DataResponse
	updateTable func()
	setViewData func(data models.DataResponse)
	syncTimer   *time.Timer
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
		tui.client = NewClient(tui.config.ServerURL)
		tui.client.SetAuthToken(tui.config.Token)
	} else {
		tui.client = NewClient(tui.config.ServerURL)
	}

	return tui, nil
}

// loadConfig загружает конфигурацию из файла
func (t *TUI) loadConfig() error {
	if _, err := os.Stat(t.configPath); os.IsNotExist(err) {
		return err
	}

	data, err := os.ReadFile(t.configPath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, t.config); err != nil {
		return err
	}

	return nil
}

// saveConfig сохраняет конфигурацию в файл
func (t *TUI) saveConfig() error {
	dir := filepath.Dir(t.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(t.config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(t.configPath, data, 0644)
}

// Run запускает TUI
func (t *TUI) Run() error {
	t.initPages()
	t.startAutoSync()

	return t.app.SetRoot(t.pages, true).EnableMouse(true).Run()
}

// startAutoSync запускает автоматическую синхронизацию данных
func (t *TUI) startAutoSync() {
	if t.syncTimer != nil {
		t.syncTimer.Stop()
	}

	t.syncTimer = time.AfterFunc(15*time.Second, func() {
		t.app.QueueUpdateDraw(func() {
			t.loadData()
			t.startAutoSync()
		})
	})
}

// stopAutoSync останавливает автоматическую синхронизацию
func (t *TUI) stopAutoSync() {
	if t.syncTimer != nil {
		t.syncTimer.Stop()
		t.syncTimer = nil
	}
}

// initPages инициализирует страницы TUI
func (t *TUI) initPages() {
	t.pages.AddPage("login", t.createLoginPage(), true, true)
	t.pages.AddPage("register", t.createRegisterPage(), true, false)
	t.pages.AddPage("main", t.createMainPage(), true, false)
	t.pages.AddPage("add", t.createAddPage(), true, false)
	t.pages.AddPage("view", t.createViewPage(), true, false)
}

// createLoginPage создает страницу входа
func (t *TUI) createLoginPage() tview.Primitive {
	loginForm := tview.NewForm()

	loginForm.AddInputField("Сервер", t.config.ServerURL, 30, nil, func(text string) {
		t.config.ServerURL = text
		t.client = NewClient(text)
	})

	loginForm.AddInputField("Имя пользователя", "", 20, nil, nil)
	loginForm.AddPasswordField("Пароль", "", 20, '*', nil)

	loginForm.AddButton("Войти", func() {
		usernameField := loginForm.GetFormItemByLabel("Имя пользователя").(*tview.InputField)
		passwordField := loginForm.GetFormItemByLabel("Пароль").(*tview.InputField)

		username := strings.ToLower(usernameField.GetText())
		password := passwordField.GetText()

		if username == "" || password == "" {
			t.showError("Имя пользователя и пароль не могут быть пустыми")
			return
		}

		_, token, err := t.client.Login(username, password)
		if err != nil {
			t.showError(fmt.Sprintf("Ошибка входа: %v", err))
			return
		}

		t.config.Token = token
		t.config.Username = username
		t.saveConfig()

		t.loadData()

		if !t.client.HasMasterPassword() {
			t.showCreateMasterPasswordDialog()
		} else {
			t.pages.SwitchToPage("main")
		}
	})

	loginForm.AddButton("Регистрация", func() {
		t.pages.SwitchToPage("register")
	})

	loginForm.SetTitle("GophKeeper - Вход").SetBorder(true)

	return loginForm
}

// createRegisterPage создает страницу регистрации
func (t *TUI) createRegisterPage() tview.Primitive {
	registerForm := tview.NewForm()

	registerForm.AddInputField("Сервер", t.config.ServerURL, 30, nil, func(text string) {
		t.config.ServerURL = text
		t.client = NewClient(text)
	})

	registerForm.AddInputField("Имя пользователя", "", 20, nil, nil)
	registerForm.AddPasswordField("Пароль", "", 20, '*', nil)
	registerForm.AddPasswordField("Подтверждение пароля", "", 20, '*', nil)

	registerForm.AddButton("Зарегистрироваться", func() {
		usernameField := registerForm.GetFormItemByLabel("Имя пользователя").(*tview.InputField)
		passwordField := registerForm.GetFormItemByLabel("Пароль").(*tview.InputField)
		confirmPasswordField := registerForm.GetFormItemByLabel("Подтверждение пароля").(*tview.InputField)

		username := strings.ToLower(usernameField.GetText())
		password := passwordField.GetText()
		confirmPassword := confirmPasswordField.GetText()

		if username == "" || password == "" {
			t.showError("Имя пользователя и пароль не могут быть пустыми")
			return
		}

		if password != confirmPassword {
			t.showError("Пароли не совпадают")
			return
		}

		_, token, err := t.client.Register(username, password)
		if err != nil {
			t.showError(fmt.Sprintf("Ошибка регистрации: %v", err))
			return
		}

		t.config.Token = token
		t.config.Username = username
		t.saveConfig()

		t.loadData()

		if !t.client.HasMasterPassword() {
			t.showCreateMasterPasswordDialog()
		} else {
			t.pages.SwitchToPage("main")
		}
	})

	registerForm.AddButton("Назад", func() {
		t.pages.SwitchToPage("login")
	})

	registerForm.SetTitle("GophKeeper - Регистрация").SetBorder(true)

	return registerForm
}

// createMainPage создает главную страницу
func (t *TUI) createMainPage() tview.Primitive {
	table := tview.NewTable().
		SetBorders(true).
		SetSelectable(true, false)

	table.SetCell(0, 0, tview.NewTableCell("ID").SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter))
	table.SetCell(0, 1, tview.NewTableCell("Тип").SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter))
	table.SetCell(0, 2, tview.NewTableCell("Имя").SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter))
	table.SetCell(0, 3, tview.NewTableCell("Обновлено").SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter))

	table.SetSelectedFunc(func(row, column int) {
		if row > 0 && row <= len(t.dataList) {
			t.setViewData(t.dataList[row-1])

			t.pages.SwitchToPage("view")
		}
	})

	buttons := tview.NewForm().
		AddButton("Добавить", func() {
			t.pages.SwitchToPage("add")
		}).
		AddButton("Обновить", func() {
			t.loadData()
		}).
		AddButton("Выход", func() {
			t.stopAutoSync()

			t.config.Token = ""
			t.saveConfig()

			t.pages.SwitchToPage("login")
		})

	info := tview.NewTextView().
		SetText(fmt.Sprintf("Пользователь: %s", t.config.Username)).
		SetTextAlign(tview.AlignCenter)

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(info, 1, 1, false).
		AddItem(table, 0, 1, true).
		AddItem(buttons, 3, 1, false)

	layout.SetTitle("GophKeeper - Главная").SetBorder(true)

	t.updateTable = func() {
		table.Clear()

		table.SetCell(0, 0, tview.NewTableCell("ID").SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter))
		table.SetCell(0, 1, tview.NewTableCell("Тип").SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter))
		table.SetCell(0, 2, tview.NewTableCell("Имя").SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter))
		table.SetCell(0, 3, tview.NewTableCell("Обновлено").SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter))

		for i, data := range t.dataList {
			row := i + 1
			table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", data.ID)).SetAlign(tview.AlignCenter))
			table.SetCell(row, 1, tview.NewTableCell(string(data.Type)).SetAlign(tview.AlignLeft))
			table.SetCell(row, 2, tview.NewTableCell(data.Name).SetAlign(tview.AlignLeft))
			table.SetCell(row, 3, tview.NewTableCell(data.UpdatedAt.Format("2006-01-02 15:04:05")).SetAlign(tview.AlignRight))
		}

		info.SetText(fmt.Sprintf("Пользователь: %s | Записей: %d | Автосинхронизация: каждые 15 сек", t.config.Username, len(t.dataList)))
	}

	return layout
}

// createAddPage создает страницу добавления данных
func (t *TUI) createAddPage() tview.Primitive {
	addForm := tview.NewForm()

	dataTypes := []string{"login_password", "text_data", "binary_data", "card_data"}

	var nameField *tview.InputField
	var currentTypeIndex int

	var createFormFields func(int)
	createFormFields = func(typeIndex int) {
		addForm.Clear(true)

		currentTypeIndex = typeIndex

		addForm.AddDropDown("Тип", dataTypes, typeIndex, func(option string, index int) {
			if index != currentTypeIndex {
				createFormFields(index)
			}
		})

		nameField = tview.NewInputField().
			SetLabel("Наименование").
			SetFieldWidth(30)
		addForm.AddFormItem(nameField)

		switch dataTypes[typeIndex] {
		case "login_password":
			loginField := tview.NewInputField().
				SetLabel("Логин").
				SetFieldWidth(30)
			passwordField := tview.NewInputField().
				SetLabel("Пароль").
				SetFieldWidth(30).
				SetMaskCharacter('*')

			addForm.AddFormItem(loginField)
			addForm.AddFormItem(passwordField)

		case "text_data":
			contentArea := tview.NewTextArea().
				SetLabel("Содержимое")
			contentArea.SetBorder(true)

			addForm.AddFormItem(contentArea)

		case "card_data":
			numberField := tview.NewInputField().
				SetLabel("Номер карты").
				SetFieldWidth(20)
			holderField := tview.NewInputField().
				SetLabel("Владелец карты").
				SetFieldWidth(30)
			expiryField := tview.NewInputField().
				SetLabel("Срок действия (MM/YY)").
				SetFieldWidth(5)
			cvvField := tview.NewInputField().
				SetLabel("CVV").
				SetFieldWidth(3)

			addForm.AddFormItem(numberField)
			addForm.AddFormItem(holderField)
			addForm.AddFormItem(expiryField)
			addForm.AddFormItem(cvvField)

		case "binary_data":
			pathField := tview.NewInputField().
				SetLabel("Путь к файлу").
				SetFieldWidth(40)

			addForm.AddFormItem(pathField)
			addForm.AddButton("Выбрать файл", func() {
				t.showFileDialog(pathField)
			})
		}

		addForm.AddButton("Добавить", func() {
			name := nameField.GetText()

			if name == "" {
				t.showError("Имя не может быть пустым")
				return
			}

			req := models.DataRequest{
				Type:     models.DataType(dataTypes[currentTypeIndex]),
				Name:     name,
				Metadata: "",
			}

			switch models.DataType(dataTypes[currentTypeIndex]) {
			case models.LoginPassword:
				loginField := addForm.GetFormItemByLabel("Логин").(*tview.InputField)
				passwordField := addForm.GetFormItemByLabel("Пароль").(*tview.InputField)
				login := loginField.GetText()
				password := passwordField.GetText()

				if login == "" || password == "" {
					t.showError("Логин и пароль не могут быть пустыми")
					return
				}

				req.Content = models.LoginPasswordData{
					Login:    login,
					Password: password,
				}

			case models.TextData:
				contentArea := addForm.GetFormItem(2).(*tview.TextArea)
				content := contentArea.GetText()

				if content == "" {
					t.showError("Содержимое не может быть пустым")
					return
				}

				req.Content = models.TextDataContent{
					Content: content,
				}

			case models.CardData:
				numberField := addForm.GetFormItemByLabel("Номер карты").(*tview.InputField)
				holderField := addForm.GetFormItemByLabel("Владелец карты").(*tview.InputField)
				expiryField := addForm.GetFormItemByLabel("Срок действия (MM/YY)").(*tview.InputField)
				cvvField := addForm.GetFormItemByLabel("CVV").(*tview.InputField)

				number := numberField.GetText()
				holder := holderField.GetText()
				expiry := expiryField.GetText()
				cvv := cvvField.GetText()

				if number == "" || holder == "" || expiry == "" || cvv == "" {
					t.showError("Все поля карты должны быть заполнены")
					return
				}

				req.Content = models.CardDataContent{
					CardNumber: number,
					CardHolder: holder,
					ExpiryDate: expiry,
					CVV:        cvv,
				}

			case models.BinaryData:
				pathField := addForm.GetFormItemByLabel("Путь к файлу").(*tview.InputField)
				filePath := pathField.GetText()

				if filePath == "" {
					t.showError("Путь к файлу не может быть пустым")
					return
				}

				fileData, err := os.ReadFile(filePath)
				if err != nil {
					t.showError(fmt.Sprintf("Ошибка чтения файла: %v", err))
					return
				}

				req.Content = fileData
			}

			masterPassword, err := t.client.GetMasterPassword()
			if err != nil {
				t.showError(fmt.Sprintf("Ошибка получения мастер-пароля: %v", err))
				return
			}

			_, err = t.client.CreateEncryptedData(req, masterPassword)
			if err != nil {
				t.showError(fmt.Sprintf("Ошибка добавления данных: %v", err))
				return
			}

			t.loadData()

			t.pages.SwitchToPage("main")
		})

		addForm.AddButton("Отмена", func() {
			t.pages.SwitchToPage("main")
		})
	}

	createFormFields(0)

	addForm.SetTitle("GophKeeper - Добавление данных").SetBorder(true)

	return addForm
}

// createViewPage создает страницу просмотра данных
func (t *TUI) createViewPage() tview.Primitive {
	text := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)

	buttons := tview.NewForm().
		AddButton("Расшифровать", func() {
			idStr := text.GetText(true)
			idStr = strings.Split(idStr, ":")[1]
			idStr = strings.Split(idStr, "\n")[0]
			id, _ := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)

			var currentData models.DataResponse
			for _, data := range t.dataList {
				if data.ID == id {
					currentData = data
					break
				}
			}

			t.showPasswordDialog(currentData)
		}).
		AddButton("Удалить", func() {
			idStr := text.GetText(true)
			idStr = strings.Split(idStr, ":")[1]
			idStr = strings.Split(idStr, "\n")[0]
			id, _ := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)

			if err := t.client.DeleteData(id); err != nil {
				t.showError(fmt.Sprintf("Ошибка удаления данных: %v", err))
				return
			}

			t.loadData()

			t.pages.SwitchToPage("main")
		}).
		AddButton("Назад", func() {
			t.pages.SwitchToPage("main")
		})

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(text, 0, 1, true).
		AddItem(buttons, 3, 1, false)

	layout.SetTitle("GophKeeper - Просмотр данных").SetBorder(true)

	t.setViewData = func(data models.DataResponse) {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("ID: %d\n", data.ID))
		sb.WriteString(fmt.Sprintf("Тип: %s\n", data.Type))
		sb.WriteString(fmt.Sprintf("Имя: %s\n", data.Name))

		switch data.Type {
		case models.LoginPassword:
			if loginPass, ok := data.Content.(models.LoginPasswordData); ok {
				sb.WriteString(fmt.Sprintf("Логин: %s\n", loginPass.Login))
				sb.WriteString(fmt.Sprintf("Пароль: %s\n", loginPass.Password))
			}
		case models.TextData:
			if textData, ok := data.Content.(models.TextDataContent); ok {
				sb.WriteString(fmt.Sprintf("Содержимое:\n%s\n", textData.Content))
			}
		case models.CardData:
			if cardData, ok := data.Content.(models.CardDataContent); ok {
				sb.WriteString(fmt.Sprintf("Номер карты: %s\n", cardData.CardNumber))
				sb.WriteString(fmt.Sprintf("Владелец: %s\n", cardData.CardHolder))
				sb.WriteString(fmt.Sprintf("Срок действия: %s\n", cardData.ExpiryDate))
				sb.WriteString(fmt.Sprintf("CVV: %s\n", cardData.CVV))
			}
		case models.BinaryData:
			sb.WriteString("Бинарные данные\n")
			if binData, ok := data.Content.([]byte); ok {
				sb.WriteString(fmt.Sprintf("Размер: %d байт\n", len(binData)))
			}
		}

		if data.Metadata != "" {
			sb.WriteString(fmt.Sprintf("Метаданные: %s\n", data.Metadata))
		}

		sb.WriteString(fmt.Sprintf("Создано: %s\n", data.CreatedAt.Format("2006-01-02 15:04:05")))
		sb.WriteString(fmt.Sprintf("Обновлено: %s\n", data.UpdatedAt.Format("2006-01-02 15:04:05")))

		text.SetText(sb.String())
	}

	return layout
}

// showPasswordDialog отображает диалог для ввода мастер-пароля
func (t *TUI) showPasswordDialog(data models.DataResponse) {
	if t.client.HasMasterPassword() {
		masterPassword, err := t.client.GetMasterPassword()
		if err == nil {
			encryptedData, err := t.client.GetEncryptedData(data.ID)
			if err != nil {
				t.showError(fmt.Sprintf("Ошибка получения зашифрованных данных: %v", err))
				return
			}

			decryptedContent, err := t.client.DecryptData(encryptedData, masterPassword)
			if err != nil {
				t.showError(fmt.Sprintf("Ошибка расшифровки данных: %v", err))
				t.showManualPasswordDialog(data)
				return
			}

			data.Content = decryptedContent
			t.setViewData(data)

			t.showInfo("Данные успешно расшифрованы")
			return
		}
	}

	t.showManualPasswordDialog(data)
}

// showManualPasswordDialog отображает диалог для ручного ввода мастер-пароля
func (t *TUI) showManualPasswordDialog(data models.DataResponse) {
	form := tview.NewForm()

	passwordField := tview.NewInputField().
		SetLabel("Пароль").
		SetMaskCharacter('*').
		SetFieldWidth(30)

	form.AddFormItem(passwordField)

	form.AddButton("Расшифровать", func() {
		password := passwordField.GetText()

		if password == "" {
			t.showError("Пароль не может быть пустым")
			return
		}

		encryptedData, err := t.client.GetEncryptedData(data.ID)
		if err != nil {
			t.showError(fmt.Sprintf("Ошибка получения зашифрованных данных: %v", err))
			return
		}

		decryptedContent, err := t.client.DecryptData(encryptedData, password)
		if err != nil {
			t.showError(fmt.Sprintf("Ошибка расшифровки данных: %v", err))
			return
		}

		data.Content = decryptedContent
		t.setViewData(data)

		t.pages.RemovePage("password_dialog")

		t.showInfo("Данные успешно расшифрованы")
	})

	form.AddButton("Отмена", func() {
		t.pages.RemovePage("password_dialog")
	})

	form.SetTitle("Расшифровка данных").SetBorder(true)

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 10, 1, true).
			AddItem(nil, 0, 1, false), 40, 1, true).
		AddItem(nil, 0, 1, false)

	t.pages.AddPage("password_dialog", modal, true, true)
}

// showCreateMasterPasswordDialog отображает диалог для создания мастер-пароля
func (t *TUI) showCreateMasterPasswordDialog() {
	form := tview.NewForm()

	passwordField := tview.NewInputField().
		SetLabel("Мастер-пароль").
		SetMaskCharacter('*').
		SetFieldWidth(30)

	confirmPasswordField := tview.NewInputField().
		SetLabel("Подтверждение").
		SetMaskCharacter('*').
		SetFieldWidth(30)

	form.AddFormItem(passwordField)
	form.AddFormItem(confirmPasswordField)

	form.AddTextView("Информация", "Мастер-пароль используется для шифрования и расшифровки ваших данных.\nЗапомните его, так как восстановление невозможно!", 0, 3, true, false)

	form.AddButton("Создать", func() {
		password := passwordField.GetText()
		confirmPassword := confirmPasswordField.GetText()

		if password == "" {
			t.showError("Пароль не может быть пустым")
			return
		}

		if password != confirmPassword {
			t.showError("Пароли не совпадают")
			return
		}

		if err := t.client.SetMasterPassword(password); err != nil {
			t.showError(fmt.Sprintf("Ошибка сохранения мастер-пароля: %v", err))
			return
		}

		t.pages.RemovePage("create_master_password_dialog")

		t.showInfo("Мастер-пароль успешно создан")

		t.pages.SwitchToPage("main")
	})

	form.SetTitle("Создание мастер-пароля").SetBorder(true)

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 15, 1, true).
			AddItem(nil, 0, 1, false), 50, 1, true).
		AddItem(nil, 0, 1, false)

	t.pages.AddPage("create_master_password_dialog", modal, true, true)
}

// showFileDialog отображает диалог выбора файла
func (t *TUI) showFileDialog(pathField *tview.InputField) {
	modal := tview.NewModal()

	fileList := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true)

	currentPath := "/"
	if home, err := os.UserHomeDir(); err == nil {
		currentPath = home
	}

	var updateFileList func(string)
	updateFileList = func(path string) {
		fileList.Clear()

		fileList.AddItem(".. (Назад)", "", 0, func() {
			parent := filepath.Dir(path)
			currentPath = parent
			updateFileList(parent)
		})

		files, err := os.ReadDir(path)
		if err != nil {
			t.showError(fmt.Sprintf("Ошибка чтения директории: %v", err))
			return
		}

		var dirs []os.DirEntry
		var regularFiles []os.DirEntry

		for _, file := range files {
			if file.IsDir() {
				dirs = append(dirs, file)
			} else {
				regularFiles = append(regularFiles, file)
			}
		}

		for _, dir := range dirs {
			dirName := dir.Name()
			fileList.AddItem(dirName+"/", "", 0, func() {
				newPath := filepath.Join(path, dirName)
				currentPath = newPath
				updateFileList(newPath)
			})
		}

		for _, file := range regularFiles {
			fileName := file.Name()
			fileList.AddItem(fileName, "", 0, func() {
				filePath := filepath.Join(path, fileName)
				pathField.SetText(filePath)
				t.pages.RemovePage("file_dialog")
			})
		}
	}

	updateFileList(currentPath)

	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	pathText := tview.NewTextView().
		SetText(currentPath).
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true)

	oldUpdateFileList := updateFileList
	updateFileList = func(path string) {
		oldUpdateFileList(path)
		pathText.SetText(path)
	}

	flex.AddItem(pathText, 1, 1, false)
	flex.AddItem(fileList, 0, 1, true)

	modal.
		SetText("Выберите файл для загрузки").
		SetBackgroundColor(tcell.ColorBlack).
		AddButtons([]string{"Отмена"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			t.pages.RemovePage("file_dialog")
		})

	page := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(flex, 0, 30, true).
			AddItem(nil, 0, 1, false), 0, 30, true).
		AddItem(nil, 0, 1, false)

	t.pages.AddPage("file_dialog", page, true, true)
}

// loadData загружает данные с сервера
func (t *TUI) loadData() {
	if t.config.Token == "" {
		return
	}

	data, err := t.client.GetAllData()
	if err != nil {
		t.showError(fmt.Sprintf("Ошибка загрузки данных: %v", err))
		return
	}

	t.dataList = data

	if t.updateTable != nil {
		t.updateTable()
	}
}

// showError отображает сообщение об ошибке
func (t *TUI) showError(message string) {
	t.showDialog("Ошибка", message, "OK", nil)
}

// showInfo отображает информационное сообщение
func (t *TUI) showInfo(message string) {
	t.showDialog("Информация", message, "OK", nil)
}

// showDialog отображает диалоговое окно
func (t *TUI) showDialog(title, message, buttonText string, callback func()) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{buttonText}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			t.pages.RemovePage("dialog")
			if callback != nil {
				callback()
			}
		})

	modal.SetTitle(title).SetBorder(true)

	t.pages.AddPage("dialog", modal, true, true)
}
