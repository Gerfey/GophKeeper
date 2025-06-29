package client

import (
	"context"
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

const (
	buttonHeight = 3

	standardFieldWidth  = 30
	shortFieldWidth     = 20
	longFieldWidth      = 50
	veryShortFieldWidth = 5
	cvvFieldWidth       = 3

	idColumn      = 0
	typeColumn    = 1
	nameColumn    = 2
	updatedColumn = 3

	modalFormHeight  = 3
	dialogWidth      = 60
	dialogHeight     = 15
	dialogTextHeight = 3

	passwordFieldWidth = 30

	syncIntervalSeconds = 15

	flexProportion0 = 0
	flexProportion1 = 1
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
	data              []models.DataResponse
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

func (t *TUI) logout() {
	t.stopAutoSync()

	t.config.Token = ""
	t.client.SetAuthToken("")
	t.dataList = nil

	if err := t.saveConfig(); err != nil {
		t.showError(fmt.Sprintf("Ошибка при сохранении конфигурации: %v", err))
	}

	t.pages.SwitchToPage("login")
}

func (t *TUI) initPages() {
	t.pages.AddPage("login", t.createLoginPage(), true, true)
	t.pages.AddPage("register", t.createRegisterPage(), true, false)
	t.pages.AddPage("main", t.createMainPage(), true, false)
	t.pages.AddPage("add", t.createAddPage(), true, false)
	t.pages.AddPage("view", t.createViewPage(), true, false)
}

func (t *TUI) createLoginPage() tview.Primitive {
	loginForm := tview.NewForm()

	loginForm.AddInputField("Сервер", t.config.ServerURL, standardFieldWidth, nil, func(text string) {
		t.config.ServerURL = text
		t.client = NewClient(text, true)
	})

	loginForm.AddInputField("Имя пользователя", "", standardFieldWidth, nil, nil)
	loginForm.AddPasswordField("Пароль", "", standardFieldWidth, '*', nil)

	loginForm.AddButton("Войти", func() {
		var username, password string

		if item := loginForm.GetFormItemByLabel("Имя пользователя"); item != nil {
			if field, ok := item.(*tview.InputField); ok {
				username = field.GetText()
			}
		}
		if item := loginForm.GetFormItemByLabel("Пароль"); item != nil {
			if field, ok := item.(*tview.InputField); ok {
				password = field.GetText()
			}
		}

		if username == "" || password == "" {
			t.showError("Имя пользователя и пароль не могут быть пустыми")

			return
		}

		err := t.client.Login(context.Background(), username, password)
		if err != nil {
			t.showError(fmt.Sprintf("Ошибка входа: %v", err))

			return
		}

		t.config.Token = t.client.GetAuthToken()
		t.config.Username = username
		if saveErr := t.saveConfig(); saveErr != nil {
			t.showError(fmt.Sprintf("Ошибка сохранения конфигурации: %v", saveErr))
		}

		t.loadData()
		t.startAutoSync()

		if !t.client.HasMasterPassword() {
			t.showCreateMasterPasswordModal()
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

func (t *TUI) createRegisterPage() tview.Primitive {
	registerForm := tview.NewForm()

	t.addRegisterFormFields(registerForm)
	t.addRegisterFormButtons(registerForm)
	t.addRegisterFormNavigationButtons(registerForm)

	return registerForm
}

func (t *TUI) addRegisterFormFields(form *tview.Form) {
	form.AddInputField("Сервер", t.config.ServerURL, standardFieldWidth, nil, func(text string) {
		t.config.ServerURL = text
		t.client = NewClient(text, true)
	})

	form.AddInputField("Имя пользователя", "", standardFieldWidth, nil, nil)
	form.AddPasswordField("Пароль", "", standardFieldWidth, '*', nil)
	form.AddPasswordField("Подтверждение пароля", "", standardFieldWidth, '*', nil)
}

func (t *TUI) addRegisterFormButtons(form *tview.Form) {
	form.AddButton("Зарегистрироваться", func() {
		formData := t.getRegisterFormData(form)

		if !t.validateRegisterFormData(formData) {
			return
		}

		t.performRegistration(formData)
	})
}

func (t *TUI) getRegisterFormData(form *tview.Form) map[string]string {
	data := make(map[string]string)

	if item := form.GetFormItemByLabel("Имя пользователя"); item != nil {
		if field, ok := item.(*tview.InputField); ok {
			data["username"] = field.GetText()
		}
	}

	if item := form.GetFormItemByLabel("Пароль"); item != nil {
		if field, ok := item.(*tview.InputField); ok {
			data["password"] = field.GetText()
		}
	}

	if item := form.GetFormItemByLabel("Подтверждение пароля"); item != nil {
		if field, ok := item.(*tview.InputField); ok {
			data["confirmPassword"] = field.GetText()
		}
	}

	return data
}

func (t *TUI) validateRegisterFormData(data map[string]string) bool {
	username := data["username"]
	password := data["password"]
	confirmPassword := data["confirmPassword"]

	if username == "" || password == "" {
		t.showError("Имя пользователя и пароль не могут быть пустыми")

		return false
	}

	if password != confirmPassword {
		t.showError("Пароли не совпадают")

		return false
	}

	return true
}

func (t *TUI) performRegistration(data map[string]string) {
	username := data["username"]
	password := data["password"]

	err := t.client.Register(context.Background(), username, password)
	if err != nil {
		t.showError(fmt.Sprintf("Ошибка регистрации: %v", err))

		return
	}

	err = t.client.Login(context.Background(), username, password)
	if err != nil {
		t.showError(fmt.Sprintf("Ошибка входа: %v", err))

		return
	}

	t.config.Username = username
	t.config.Token = t.client.GetAuthToken()
	if saveErr := t.saveConfig(); saveErr != nil {
		t.showError(fmt.Sprintf("Ошибка сохранения конфигурации: %v", saveErr))

		return
	}

	t.showCreateMasterPasswordModal()

	t.pages.SwitchToPage("main")
}

func (t *TUI) addRegisterFormNavigationButtons(form *tview.Form) {
	form.AddButton("Назад", func() {
		t.pages.SwitchToPage("login")
	})
}

//nolint:funlen
func (t *TUI) createMainPage() tview.Primitive {
	t.dataTable = tview.NewTable().
		SetBorders(true).
		SetSelectable(true, false)

	t.dataTable.SetCell(
		0,
		idColumn,
		tview.NewTableCell("ID").SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter),
	)
	t.dataTable.SetCell(
		0,
		typeColumn,
		tview.NewTableCell("Тип").SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter),
	)
	t.dataTable.SetCell(
		0,
		nameColumn,
		tview.NewTableCell("Имя").SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter),
	)
	t.dataTable.SetCell(
		0,
		updatedColumn,
		tview.NewTableCell("Обновлено").SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter),
	)

	buttons := tview.NewForm().
		AddButton("Добавить", func() {
			t.pages.SwitchToPage("add")
		}).
		AddButton("Просмотр", func() {
			row, _ := t.dataTable.GetSelection()

			if row <= 0 {
				t.showError("Выберите запись для просмотра")

				return
			}

			dataIndex := row - 1
			if dataIndex >= len(t.dataList) {
				t.showError("Выберите запись для просмотра")

				return
			}

			selectedData := t.dataList[dataIndex]

			buttons := tview.NewForm()

			buttons.AddButton("Расшифровать", func() {
				t.handleDecryptData(selectedData.ID)
			})

			buttons.AddButton("Удалить", func() {
				t.handleDeleteData(selectedData.ID)
			})

			buttons.AddButton("Назад", func() {
				t.pages.SwitchToPage("main")
			})

			textView := tview.NewTextView().
				SetDynamicColors(true).
				SetRegions(true).
				SetWordWrap(true)

			fmt.Fprintf(textView, "[yellow]ID:[white] %d\n", selectedData.ID)
			fmt.Fprintf(textView, "[yellow]Тип:[white] %s\n", selectedData.Type)
			fmt.Fprintf(textView, "[yellow]Имя:[white] %s\n", selectedData.Name)
			fmt.Fprintf(textView, "[yellow]Обновлено:[white] %s\n\n", formatTime(selectedData.UpdatedAt))

			t.displayDataContentByType(textView, selectedData)

			layout := tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(textView, 0, 1, true).
				AddItem(buttons, buttonHeight, 1, false)

			layout.SetTitle("GophKeeper - Просмотр данных").SetBorder(true)

			t.pages.RemovePage("view")
			t.pages.AddPage("view", layout, true, true)

			t.pages.SwitchToPage("view")
		}).
		AddButton("Выйти из аккаунта", func() {
			t.logout()
		}).
		AddButton("Выход", func() {
			t.app.Stop()
		})

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.dataTable, 0, 1, true).
		AddItem(buttons, buttonHeight, 1, false)

	layout.SetTitle("GophKeeper - Главная").SetBorder(true)

	t.updateTable = func() {
		t.updateDataTable(t.dataList)
	}

	return layout
}

func (t *TUI) updateDataTable(data []models.DataResponse) {
	t.dataTable.Clear()
	t.dataTable.SetCell(
		0,
		idColumn,
		tview.NewTableCell("ID").SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter),
	)
	t.dataTable.SetCell(
		0,
		typeColumn,
		tview.NewTableCell("Тип").SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter),
	)
	t.dataTable.SetCell(
		0,
		nameColumn,
		tview.NewTableCell("Имя").SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter),
	)
	t.dataTable.SetCell(
		0,
		updatedColumn,
		tview.NewTableCell("Обновлено").SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter),
	)

	for i, item := range data {
		row := i + 1
		t.dataTable.SetCell(
			row,
			idColumn,
			tview.NewTableCell(strconv.FormatInt(item.ID, 10)).SetAlign(tview.AlignRight),
		)
		t.dataTable.SetCell(row, typeColumn, tview.NewTableCell(string(item.Type)).SetAlign(tview.AlignLeft))
		t.dataTable.SetCell(row, nameColumn, tview.NewTableCell(item.Name).SetAlign(tview.AlignLeft))
		t.dataTable.SetCell(
			row,
			updatedColumn,
			tview.NewTableCell(item.UpdatedAt.Format("02.01.2006 15:04:05")).SetAlign(tview.AlignLeft),
		)
	}

	t.data = data
	t.dataList = data
}

func (t *TUI) createAddPage() tview.Primitive {
	addForm := tview.NewForm()
	dataTypes := []string{"login_password", "text_data", "card_data", "binary_data"}
	var nameField *tview.InputField
	currentTypeIndex := 0

	var addFormFields func()
	addFormFields = func() {
		addForm.Clear(true)

		addForm.AddDropDown("Тип", dataTypes, currentTypeIndex, func(_ string, index int) {
			if index != currentTypeIndex {
				currentTypeIndex = index
				addFormFields()
			}
		})

		nameField = t.addNameField(addForm)

		t.addDataTypeSpecificFields(addForm, dataTypes[currentTypeIndex])
		t.addAddPageButtons(addForm, nameField, dataTypes, currentTypeIndex)
	}

	addFormFields()

	addForm.SetTitle("GophKeeper - Добавление данных").SetBorder(true)

	return addForm
}

func (t *TUI) addNameField(form *tview.Form) *tview.InputField {
	nameField := tview.NewInputField().
		SetLabel("Наименование").
		SetFieldWidth(standardFieldWidth)

	form.AddFormItem(nameField)

	return nameField
}

func (t *TUI) addDataTypeSpecificFields(form *tview.Form, dataType string) {
	switch dataType {
	case "login_password":
		t.addLoginPasswordFields(form)
	case "text_data":
		t.addTextDataFields(form)
	case "card_data":
		t.addCardDataFields(form)
	case "binary_data":
		t.addBinaryDataFields(form)
	}
}

func (t *TUI) addLoginPasswordFields(form *tview.Form) {
	loginField := tview.NewInputField().
		SetLabel("Логин").
		SetFieldWidth(standardFieldWidth)
	passwordField := tview.NewInputField().
		SetLabel("Пароль").
		SetFieldWidth(standardFieldWidth).
		SetMaskCharacter('*')

	form.AddFormItem(loginField)
	form.AddFormItem(passwordField)
}

func (t *TUI) addTextDataFields(form *tview.Form) {
	contentArea := tview.NewTextArea().
		SetLabel("Текст")
	contentArea.SetBorder(true)

	form.AddFormItem(contentArea)
}

func (t *TUI) addCardDataFields(form *tview.Form) {
	numberField := tview.NewInputField().
		SetLabel("Номер карты").
		SetFieldWidth(shortFieldWidth)
	holderField := tview.NewInputField().
		SetLabel("Владелец карты").
		SetFieldWidth(standardFieldWidth)
	expiryField := tview.NewInputField().
		SetLabel("Срок действия (MM/YY)").
		SetFieldWidth(veryShortFieldWidth)
	cvvField := tview.NewInputField().
		SetLabel("CVV").
		SetFieldWidth(cvvFieldWidth)

	form.AddFormItem(numberField)
	form.AddFormItem(holderField)
	form.AddFormItem(expiryField)
	form.AddFormItem(cvvField)
}

func (t *TUI) addBinaryDataFields(form *tview.Form) {
	pathField := tview.NewInputField().
		SetLabel("Путь к файлу").
		SetFieldWidth(longFieldWidth)

	form.AddFormItem(pathField)

	form.AddButton("Обзор файла", func() {
		t.showFileDialog(func(filePath string) {
			pathField.SetText(filePath)
		})
	})
}

func (t *TUI) addAddPageButtons(
	form *tview.Form,
	nameField *tview.InputField,
	dataTypes []string,
	currentTypeIndex int,
) {
	form.AddButton("Добавить", func() {
		t.handleAddData(form, nameField, dataTypes, currentTypeIndex)
	})

	form.AddButton("Назад", func() {
		t.pages.SwitchToPage("main")
	})
}

func (t *TUI) handleAddData(form *tview.Form, nameField *tview.InputField, dataTypes []string, currentTypeIndex int) {
	name := nameField.GetText()
	if name == "" {
		t.showError("Необходимо указать наименование")

		return
	}

	if !t.client.HasMasterPassword() {
		t.showCreateMasterPasswordModal()

		return
	}

	t.showPasswordDialog(func(masterPassword string) {
		req := &models.DataRequest{
			Type: models.DataType(dataTypes[currentTypeIndex]),
			Name: name,
		}

		t.processDataByType(form, req, dataTypes[currentTypeIndex])

		encryptedData, err := t.client.EncryptData(req.Content, req.Type, masterPassword)
		if err != nil {
			t.showError(fmt.Sprintf("Ошибка шифрования данных: %v", err))

			return
		}

		_, err = t.client.CreateData(context.Background(), req.Name, req.Type, encryptedData)
		if err != nil {
			t.showError(fmt.Sprintf("Ошибка создания данных: %v", err))

			return
		}

		t.loadData()

		t.pages.SwitchToPage("main")

		t.showInfo("Данные успешно добавлены")
	})
}

//nolint:gocognit,funlen
func (t *TUI) processDataByType(form *tview.Form, req *models.DataRequest, dataType string) {
	switch dataType {
	case string(models.LoginPassword):
		login := ""
		password := ""

		if item := form.GetFormItemByLabel("Логин"); item != nil {
			if field, ok := item.(*tview.InputField); ok {
				login = field.GetText()
			}
		}

		if item := form.GetFormItemByLabel("Пароль"); item != nil {
			if field, ok := item.(*tview.InputField); ok {
				password = field.GetText()
			}
		}

		loginData := models.LoginPasswordData{
			Login:    login,
			Password: password,
		}

		req.Content = loginData

	case string(models.TextData):
		text := ""

		if item := form.GetFormItemByLabel("Текст"); item != nil {
			if field, ok := item.(*tview.TextArea); ok {
				text = field.GetText()
			}
		}

		textData := models.TextDataContent{
			Text: text,
		}

		req.Content = textData

	case string(models.CardData):
		cardNumber := ""
		cardHolder := ""
		expiryDate := ""
		cvv := ""

		if item := form.GetFormItemByLabel("Номер карты"); item != nil {
			if field, ok := item.(*tview.InputField); ok {
				cardNumber = field.GetText()
			}
		}

		if item := form.GetFormItemByLabel("Владелец карты"); item != nil {
			if field, ok := item.(*tview.InputField); ok {
				cardHolder = field.GetText()
			}
		}

		if item := form.GetFormItemByLabel("Срок действия (MM/YY)"); item != nil {
			if field, ok := item.(*tview.InputField); ok {
				expiryDate = field.GetText()
			}
		}

		if item := form.GetFormItemByLabel("CVV"); item != nil {
			if field, ok := item.(*tview.InputField); ok {
				cvv = field.GetText()
			}
		}

		cardData := models.CardDataContent{
			CardNumber: cardNumber,
			CardHolder: cardHolder,
			ExpiryDate: expiryDate,
			CVV:        cvv,
		}

		req.Content = cardData

	case string(models.BinaryData):
		filePath := ""

		if item := form.GetFormItemByLabel("Путь к файлу"); item != nil {
			if field, ok := item.(*tview.InputField); ok {
				filePath = field.GetText()
			}
		}

		fileData, err := os.ReadFile(filePath)
		if err != nil {
			t.showError(fmt.Sprintf("Ошибка чтения файла: %v", err))

			return
		}

		fileName := filepath.Base(filePath)

		binaryData := struct {
			FileName string `json:"file_name"`
			Data     []byte `json:"data"`
		}{
			FileName: fileName,
			Data:     fileData,
		}

		req.Content = binaryData
	}
}

func (t *TUI) createViewPage() tview.Primitive {
	text := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)

	buttons := t.createViewPageButtons(text)

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(text, 0, 1, true).
		AddItem(buttons, buttonHeight, 1, false)

	layout.SetTitle("GophKeeper - Просмотр данных").SetBorder(true)

	t.setViewData = func(data models.DataResponse) {
		t.displayDataInTextView(text, data)
	}

	return layout
}

func (t *TUI) createViewPageButtons(text *tview.TextView) *tview.Form {
	buttons := tview.NewForm()

	buttons.AddButton("Расшифровать", func() {
		id := t.extractIDFromText(text)
		t.handleDecryptData(id)
	})

	buttons.AddButton("Удалить", func() {
		id := t.extractIDFromText(text)
		t.handleDeleteData(id)
	})

	buttons.AddButton("Назад", func() {
		t.pages.SwitchToPage("main")
	})

	return buttons
}

func (t *TUI) extractIDFromText(text *tview.TextView) int64 {
	idStr := text.GetText(true)
	idStr = strings.Split(idStr, ":")[1]
	idStr = strings.Split(idStr, "\n")[0]
	id, _ := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)

	return id
}

//nolint:gocognit
func (t *TUI) handleDecryptData(id int64) {
	var currentData models.DataResponse
	for _, data := range t.dataList {
		if data.ID == id {
			currentData = data

			break
		}
	}

	t.showPasswordDialog(func(password string) {
		encryptedData, err := t.client.GetEncryptedData(id)
		if err != nil {
			t.showError(fmt.Sprintf("Ошибка получения зашифрованных данных: %v", err))

			return
		}

		decryptedContent, err := t.client.DecryptData(encryptedData, password)
		if err != nil {
			t.showError(fmt.Sprintf("Ошибка расшифровки данных: %v", err))

			return
		}

		for i, data := range t.dataList {
			if data.ID == id {
				t.dataList[i].Content = decryptedContent
				currentData = t.dataList[i]

				break
			}
		}

		buttons := tview.NewForm()

		buttons.AddButton("Расшифровать", func() {
			t.handleDecryptData(id)
		})

		if currentData.Type == models.BinaryData && currentData.Content != nil {
			if binaryData, ok := currentData.Content.(models.BinaryDataContent); ok && len(binaryData.Data) > 0 {
				buttons.AddButton("Сохранить файл", func() {
					t.showFileDialog(func(filePath string) {
						writeErr := os.WriteFile(filePath, binaryData.Data, 0600)
						if writeErr != nil {
							t.showError(fmt.Sprintf("Ошибка при сохранении файла: %v", writeErr))
						} else {
							t.showInfo(fmt.Sprintf("Файл успешно сохранен: %s", filePath))
						}
					})
				})
			}
		}

		buttons.AddButton("Удалить", func() {
			t.handleDeleteData(id)
		})

		buttons.AddButton("Назад", func() {
			t.pages.SwitchToPage("main")
		})

		newTextView := tview.NewTextView().
			SetDynamicColors(true).
			SetRegions(true).
			SetWordWrap(true)

		fmt.Fprintf(newTextView, "[yellow]ID:[white] %d\n", currentData.ID)
		fmt.Fprintf(newTextView, "[yellow]Тип:[white] %s\n", currentData.Type)
		fmt.Fprintf(newTextView, "[yellow]Имя:[white] %s\n", currentData.Name)
		fmt.Fprintf(newTextView, "[yellow]Обновлено:[white] %s\n\n", formatTime(currentData.UpdatedAt))

		t.displayDataContentByType(newTextView, currentData)

		newLayout := tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(newTextView, 0, 1, true).
			AddItem(buttons, buttonHeight, 1, false)

		newLayout.SetTitle("GophKeeper - Просмотр данных").SetBorder(true)

		t.pages.RemovePage("view")
		t.pages.AddPage("view", newLayout, true, true)

		t.showInfo("Данные успешно расшифрованы")
	})
}

func (t *TUI) handleDeleteData(id int64) {
	if err := t.client.DeleteData(context.Background(), id); err != nil {
		t.showError(fmt.Sprintf("Ошибка удаления данных: %v", err))

		return
	}

	t.loadData()

	t.pages.SwitchToPage("main")
}

func (t *TUI) displayDataInTextView(text *tview.TextView, data models.DataResponse) {
	text.Clear()

	fmt.Fprintf(text, "[yellow]ID:[white] %d\n", data.ID)
	fmt.Fprintf(text, "[yellow]Тип:[white] %s\n", data.Type)
	fmt.Fprintf(text, "[yellow]Имя:[white] %s\n", data.Name)
	fmt.Fprintf(text, "[yellow]Обновлено:[white] %s\n\n", formatTime(data.UpdatedAt))

	t.displayDataContentByType(text, data)
}

func (t *TUI) displayDataContentByType(text *tview.TextView, data models.DataResponse) {
	if data.Content == nil {
		fmt.Fprintf(text, "[yellow]Содержимое:[white] Зашифровано\n")

		return
	}

	switch data.Type {
	case models.LoginPassword:
		t.displayLoginPasswordContent(text, data.Content)
	case models.TextData:
		t.displayTextDataContent(text, data.Content)
	case models.CardData:
		t.displayCardDataContent(text, data.Content)
	case models.BinaryData:
		t.displayBinaryDataContent(text, data.Content)
	default:
		fmt.Fprintf(text, "[yellow]Содержимое:[white] Неизвестный тип данных\n")
	}
}

func (t *TUI) displayLoginPasswordContent(text *tview.TextView, content interface{}) {
	if loginData, ok := content.(models.LoginPasswordData); ok {
		fmt.Fprintf(text, "[yellow]Логин:[white] %s\n", loginData.Login)
		fmt.Fprintf(text, "[yellow]Пароль:[white] %s\n", loginData.Password)
	} else {
		fmt.Fprintf(text, "[red]Ошибка:[white] Не удалось преобразовать данные в формат логин/пароль\n")
		fmt.Fprintf(text, "[yellow]Отладочная информация:[white] %v\n", content)
	}
}

func (t *TUI) displayTextDataContent(text *tview.TextView, content interface{}) {
	if textData, ok := content.(models.TextDataContent); ok {
		switch {
		case textData.Text != "":
			fmt.Fprintf(text, "[yellow]Текст:[white]\n%s\n", textData.Text)
		case textData.Content != "":
			fmt.Fprintf(text, "[yellow]Текст:[white]\n%s\n", textData.Content)
		default:
			fmt.Fprintf(text, "[yellow]Текст:[white] [red](пусто)[white]\n")
		}
	} else {
		if textStr, isString := content.(string); isString {
			fmt.Fprintf(text, "[yellow]Текст:[white]\n%s\n", textStr)
		} else {
			fmt.Fprintf(text, "[red]Ошибка:[white] Не удалось преобразовать данные в текстовый формат\n")
			fmt.Fprintf(text, "[yellow]Отладочная информация:[white] %v\n", content)
		}
	}
}

func (t *TUI) displayCardDataContent(text *tview.TextView, content interface{}) {
	if cardData, ok := content.(models.CardDataContent); ok {
		fmt.Fprintf(text, "[yellow]Номер карты:[white] %s\n", cardData.CardNumber)
		fmt.Fprintf(text, "[yellow]Владелец:[white] %s\n", cardData.CardHolder)
		fmt.Fprintf(text, "[yellow]Срок действия:[white] %s\n", cardData.ExpiryDate)
		fmt.Fprintf(text, "[yellow]CVV:[white] %s\n", cardData.CVV)
	} else {
		fmt.Fprintf(text, "[red]Ошибка:[white] Не удалось преобразовать данные в формат карты\n")
		fmt.Fprintf(text, "[yellow]Отладочная информация:[white] %v\n", content)
	}
}

func (t *TUI) displayBinaryDataContent(text *tview.TextView, content interface{}) {
	if t.displayStandardBinaryContent(text, content) {
		return
	}

	if t.displayStructBinaryContent(text, content) {
		return
	}

	fmt.Fprintf(text, "[red]Ошибка:[white] Не удалось преобразовать данные в формат бинарного файла\n")
	fmt.Fprintf(text, "[yellow]Отладочная информация:[white] %v\n", content)
}

func (t *TUI) displayStandardBinaryContent(text *tview.TextView, content interface{}) bool {
	binaryData, ok := content.(models.BinaryDataContent)
	if !ok {
		return false
	}

	fmt.Fprintf(text, "[yellow]Имя файла:[white] %s\n", binaryData.FileName)
	if len(binaryData.Data) > 0 {
		fmt.Fprintf(text, "[yellow]Размер:[white] %d байт\n", len(binaryData.Data))
	}

	return true
}

func (t *TUI) displayStructBinaryContent(text *tview.TextView, content interface{}) bool {
	binaryData, isStruct := content.(struct {
		FileName string `json:"file_name"`
		Data     []byte `json:"data"`
	})
	if !isStruct {
		return false
	}

	fmt.Fprintf(text, "[yellow]Имя файла:[white] %s\n", binaryData.FileName)
	if len(binaryData.Data) > 0 {
		fmt.Fprintf(text, "[yellow]Размер:[white] %d байт\n", len(binaryData.Data))
	}

	return true
}

func (t *TUI) loadData() {
	if t.config.Token == "" {
		t.showError("Необходимо войти в систему")

		return
	}

	data, err := t.client.GetAllData()
	if err != nil {
		t.showError(fmt.Sprintf("Ошибка загрузки данных: %v", err))

		return
	}

	t.updateDataTable(data)
}

func (t *TUI) showError(message string) {
	t.showDialog("Ошибка", message, "OK", nil)
}

func (t *TUI) showInfo(message string) {
	t.showDialog("Информация", message, "OK", nil)
}

func (t *TUI) showDialog(title, message, buttonText string, callback func()) {
	text := tview.NewTextView().
		SetText(message).
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	form := tview.NewForm().
		AddButton(buttonText, func() {
			t.pages.RemovePage("dialog")

			if callback != nil {
				callback()
			}
		})
	form.SetButtonsAlign(tview.AlignCenter)

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, flexProportion0, flexProportion1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, flexProportion0, flexProportion1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(text, dialogTextHeight, flexProportion1, false).
				AddItem(form, modalFormHeight, flexProportion0, true), dialogWidth, flexProportion1, true).
			AddItem(nil, flexProportion0, flexProportion1, false), dialogHeight, flexProportion1, true).
		AddItem(nil, flexProportion0, flexProportion1, false)

	modal.SetBorder(true).
		SetTitle(title).
		SetTitleAlign(tview.AlignCenter)

	t.pages.AddPage("dialog", modal, true, true)
}

func (t *TUI) showCreateMasterPasswordModal() {
	form := tview.NewForm()

	form.AddPasswordField("Мастер-пароль:", "", passwordFieldWidth, '*', nil)
	form.AddPasswordField("Подтверждение мастер-пароля:", "", passwordFieldWidth, '*', nil)

	form.AddButton("OK", func() {
		password := ""
		confirmPassword := ""

		if item := form.GetFormItemByLabel("Мастер-пароль:"); item != nil {
			if field, ok := item.(*tview.InputField); ok {
				password = field.GetText()
			}
		}

		if item := form.GetFormItemByLabel("Подтверждение мастер-пароля:"); item != nil {
			if field, ok := item.(*tview.InputField); ok {
				confirmPassword = field.GetText()
			}
		}

		if password == "" {
			t.showError("Мастер-пароль не может быть пустым")

			return
		}

		if password != confirmPassword {
			t.showError("Пароли не совпадают")

			return
		}

		if err := t.client.SetMasterPassword(password); err != nil {
			t.showError(fmt.Sprintf("Ошибка установки мастер-пароля: %v", err))

			return
		}

		t.config.HasMasterPassword = true
		if saveErr := t.saveConfig(); saveErr != nil {
			t.showError(fmt.Sprintf("Ошибка сохранения конфигурации: %v", saveErr))

			return
		}

		t.pages.RemovePage("create_master_password")

		t.showInfo("Мастер-пароль успешно установлен")

		t.pages.SwitchToPage("main")
	})

	form.AddButton("Отмена", func() {
		t.pages.RemovePage("create_master_password")
	})

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, flexProportion0, flexProportion1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, flexProportion0, flexProportion1, false).
			AddItem(form, dialogWidth, flexProportion1, true).
			AddItem(nil, flexProportion0, flexProportion1, false), dialogHeight, flexProportion1, true).
		AddItem(nil, flexProportion0, flexProportion1, false)

	modal.SetBorder(true).
		SetTitle("Создание мастер-пароля").
		SetTitleAlign(tview.AlignCenter)

	t.pages.AddPage("create_master_password", modal, true, true)
}

func (t *TUI) showPasswordDialogFixed(callback func(password string)) {
	form := tview.NewForm()

	form.AddPasswordField("Мастер-пароль:", "", passwordFieldWidth, '*', nil)

	form.AddButton("OK", func() {
		password := ""
		if item := form.GetFormItem(0); item != nil {
			if field, ok := item.(*tview.InputField); ok {
				password = field.GetText()
			}
		}

		if password == "" {
			t.showError("Мастер-пароль не может быть пустым")

			return
		}

		if !t.config.HasMasterPassword {
			if err := t.client.SetMasterPassword(password); err != nil {
				t.showError(fmt.Sprintf("Ошибка установки мастер-пароля: %v", err))

				return
			}

			t.config.HasMasterPassword = true
			if saveErr := t.saveConfig(); saveErr != nil {
				t.showError(fmt.Sprintf("Ошибка сохранения конфигурации: %v", saveErr))

				return
			}
		} else if !t.client.VerifyMasterPassword(password) {
			t.showError("Неверный мастер-пароль")

			return
		}

		t.pages.RemovePage("password_dialog")

		callback(password)
	})

	form.AddButton("Отмена", func() {
		t.pages.RemovePage("password_dialog")
	})

	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, flexProportion0, flexProportion1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, flexProportion0, flexProportion1, false).
			AddItem(form, dialogWidth, flexProportion1, true).
			AddItem(nil, flexProportion0, flexProportion1, false), dialogHeight, flexProportion1, true).
		AddItem(nil, flexProportion0, flexProportion1, false)

	modal.SetBorder(true).
		SetTitle("Мастер-пароль").
		SetTitleAlign(tview.AlignCenter)

	t.pages.AddPage("password_dialog", modal, true, true)
}

func (t *TUI) showPasswordDialog(callback func(password string)) {
	t.showPasswordDialogFixed(callback)
}

//nolint:funlen
func (t *TUI) showFileDialog(callback func(filePath string)) {
	var currentDir string

	homeDir, err := os.UserHomeDir()
	if err == nil {
		currentDir = homeDir
	} else {
		var err2 error
		currentDir, err2 = os.Getwd()
		if err2 != nil {
			t.showError(fmt.Sprintf("Ошибка получения текущей директории: %v", err2))

			return
		}
	}

	var updateFileList func(string)
	updateFileList = func(dir string) {
		t.lastFileDialogDir = dir

		files, readErr := os.ReadDir(dir)
		if readErr != nil {
			t.showError(fmt.Sprintf("Ошибка чтения директории: %v", readErr))

			return
		}

		list := tview.NewList().ShowSecondaryText(false)

		list.AddItem("..", "", 0, func() {
			parentDir := filepath.Dir(dir)
			t.pages.RemovePage("file_dialog")
			updateFileList(parentDir)
		})

		for _, file := range files {
			fileName := file.Name()
			isDir := file.IsDir()

			if strings.HasPrefix(fileName, ".") {
				continue
			}

			localFileName := fileName

			if isDir {
				list.AddItem(fileName+"/", "", 0, func() {
					path := filepath.Join(dir, localFileName)

					t.pages.RemovePage("file_dialog")
					updateFileList(path)
				})
			} else {
				list.AddItem(fileName, "", 0, func() {
					path := filepath.Join(dir, localFileName)
					t.pages.RemovePage("file_dialog")
					callback(path)
				})
			}
		}

		inputField := tview.NewInputField().
			SetLabel("Имя файла: ").
			SetFieldWidth(standardFieldWidth)

		form := tview.NewForm()

		form.AddButton("Сохранить", func() {
			fileName := inputField.GetText()
			if fileName == "" {
				t.showError("Введите имя файла")

				return
			}

			path := filepath.Join(dir, fileName)

			t.pages.RemovePage("file_dialog")
			callback(path)
		})

		form.AddButton("Отмена", func() {
			t.pages.RemovePage("file_dialog")
		})

		form.SetButtonsAlign(tview.AlignCenter)

		layout := tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(tview.NewTextView().SetText("Текущая директория: "+dir), 1, 0, false).
			AddItem(list, 0, 1, true).
			AddItem(inputField, 1, 0, false).
			AddItem(form, modalFormHeight, 0, false)

		modal := tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(nil, 0, 1, false).
				AddItem(layout, dialogWidth, 1, true).
				AddItem(nil, 0, 1, false), dialogHeight, 1, true).
			AddItem(nil, 0, 1, false)

		modal.SetBorder(true).
			SetTitle("Выберите файл").
			SetTitleAlign(tview.AlignCenter)

		t.pages.AddPage("file_dialog", modal, true, true)

		t.app.SetFocus(list)
	}

	updateFileList(currentDir)
}

func formatTime(t time.Time) string {
	return t.Format("02.01.2006 15:04:05")
}
