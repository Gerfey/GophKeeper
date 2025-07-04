package client

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/gerfey/gophkeeper/internal/models"
)

func (t *TUI) updateDataTable(data []models.DataResponse) {
	table := t.dataTable
	table.Clear()

	table.SetCell(0, idColumn, tview.NewTableCell("ID").SetTextColor(tcell.ColorYellow).SetSelectable(false))
	table.SetCell(0, typeColumn, tview.NewTableCell("Тип").SetTextColor(tcell.ColorYellow).SetSelectable(false))
	table.SetCell(0, nameColumn, tview.NewTableCell("Название").SetTextColor(tcell.ColorYellow).SetSelectable(false))
	table.SetCell(
		0,
		updatedColumn,
		tview.NewTableCell("Обновлено").SetTextColor(tcell.ColorYellow).SetSelectable(false),
	)

	for i, item := range data {
		row := i + 1
		table.SetCell(row, idColumn, tview.NewTableCell(strconv.FormatInt(item.ID, 10)))
		table.SetCell(row, typeColumn, tview.NewTableCell(t.getDataTypeLabel(item.Type)))
		table.SetCell(row, nameColumn, tview.NewTableCell(item.Name))
		table.SetCell(row, updatedColumn, tview.NewTableCell(formatTime(item.UpdatedAt)))
	}
}

func (t *TUI) addNameField(form *tview.Form) *tview.InputField {
	nameField := tview.NewInputField()
	form.AddInputField("Название", "", standardFieldWidth, nil, func(text string) {
		nameField.SetText(text)
	})

	return nameField
}

func (t *TUI) addDataTypeSpecificFields(form *tview.Form, dataType string) {
	switch dataType {
	case "Логин/Пароль":
		t.addLoginPasswordFields(form)
	case "Текст":
		t.addTextDataFields(form)
	case "Карта":
		t.addCardDataFields(form)
	case "Файл":
		t.addBinaryDataFields(form)
	}
}

func (t *TUI) addLoginPasswordFields(form *tview.Form) {
	form.AddInputField("Логин:", "", standardFieldWidth, nil, nil)
	form.AddPasswordField("Пароль:", "", standardFieldWidth, '*', nil)
}

func (t *TUI) addTextDataFields(form *tview.Form) {
	form.AddTextArea("Текст:", "", longFieldWidth, textAreaHeight, 0, nil)
}

func (t *TUI) addCardDataFields(form *tview.Form) {
	form.AddInputField("Номер карты:", "", standardFieldWidth, nil, nil)
	form.AddInputField("Имя владельца:", "", standardFieldWidth, nil, nil)
	form.AddInputField("Срок действия (MM/YY):", "", shortFieldWidth, nil, nil)
	form.AddInputField("CVV:", "", cvvFieldWidth, nil, nil)
}

func (t *TUI) addBinaryDataFields(form *tview.Form) {
	filePathView := tview.NewTextView().
		SetText("Файл не выбран").
		SetTextColor(tcell.ColorGray)

	form.AddFormItem(filePathView)

	form.AddButton("Выбрать файл", func() {
		t.showFileDialog(func(filePath string) {
			filePathView.SetText(filePath).SetTextColor(tcell.ColorWhite)

			found := false
			for i := range form.GetFormItemCount() {
				if item, ok := form.GetFormItem(i).(*tview.InputField); ok {
					if item.GetLabel() == "FilePath" {
						item.SetText(filePath)
						found = true

						break
					}
				}
			}

			if !found {
				hiddenField := tview.NewInputField().
					SetLabel("FilePath").
					SetText(filePath)

				form.AddFormItem(hiddenField)
			}
		})
	})
}

func (t *TUI) addAddPageButtons(
	form *tview.Form,
	nameField *tview.InputField,
	dataTypes []string,
	currentTypeIndex int,
) {
	form.AddButton("Сохранить", func() {
		t.handleSaveData(form, nameField, dataTypes, currentTypeIndex)
	})

	form.AddButton("Отмена", func() {
		t.pages.SwitchToPage("main")
	})
}

func (t *TUI) handleSaveData(form *tview.Form, nameField *tview.InputField, dataTypes []string, currentTypeIndex int) {
	name := nameField.GetText()
	if name == "" {
		t.showError("Название не может быть пустым")

		return
	}

	dataType := dataTypes[currentTypeIndex]

	req := &models.DataRequest{
		Name: name,
	}

	switch dataType {
	case "Логин/Пароль":
		req.Type = models.LoginPassword
	case "Текст":
		req.Type = models.TextData
	case "Карта":
		req.Type = models.CardData
	case "Файл":
		req.Type = models.BinaryData
	}

	t.processDataByType(form, req)
}

func (t *TUI) encryptAndSaveData(data any, req *models.DataRequest) {
	t.showPasswordDialog(func(password string) {
		encryptedData, errEncryptData := t.client.EncryptData(data, req.Type, password)
		if errEncryptData != nil {
			t.showError(fmt.Sprintf("Ошибка шифрования: %v", errEncryptData))

			return
		}

		id, errCreateData := t.client.CreateData(context.Background(), req.Name, req.Type, encryptedData)
		if errCreateData != nil {
			t.showError(fmt.Sprintf("Ошибка сохранения: %v", errCreateData))

			return
		}

		t.showInfo(fmt.Sprintf("Данные успешно сохранены с ID: %d", id))
		t.loadData()
		t.pages.SwitchToPage("main")
	})
}

func (t *TUI) extractLoginPasswordData(form *tview.Form) models.LoginPasswordData {
	login := ""
	password := ""

	for i := range form.GetFormItemCount() {
		item := form.GetFormItem(i)
		if field, ok := item.(*tview.InputField); ok {
			switch field.GetLabel() {
			case "Логин:":
				login = field.GetText()
			case "Пароль:":
				password = field.GetText()
			}
		}
	}

	return models.LoginPasswordData{
		Login:    login,
		Password: password,
	}
}

func (t *TUI) extractTextData(form *tview.Form) models.TextDataContent {
	text := ""

	for i := range form.GetFormItemCount() {
		item := form.GetFormItem(i)
		if field, ok := item.(*tview.TextArea); ok {
			if field.GetLabel() == "Текст:" {
				text = field.GetText()
			}
		}
	}

	return models.TextDataContent{
		Content: text,
		Text:    text,
	}
}

func (t *TUI) extractCardData(form *tview.Form) models.CardDataContent {
	cardNumber := ""
	cardholderName := ""
	expiryDate := ""
	cvv := ""

	for i := range form.GetFormItemCount() {
		item := form.GetFormItem(i)
		if field, ok := item.(*tview.InputField); ok {
			switch field.GetLabel() {
			case "Номер карты:":
				cardNumber = field.GetText()
			case "Имя владельца:":
				cardholderName = field.GetText()
			case "Срок действия (MM/YY):":
				expiryDate = field.GetText()
			case "CVV:":
				cvv = field.GetText()
			}
		}
	}

	return models.CardDataContent{
		CardNumber: cardNumber,
		CardHolder: cardholderName,
		ExpiryDate: expiryDate,
		CVV:        cvv,
	}
}

func (t *TUI) extractBinaryData(form *tview.Form) models.BinaryDataContent {
	fileName := ""
	filePath := ""
	var fileData []byte

	for i := range form.GetFormItemCount() {
		item := form.GetFormItem(i)
		if field, ok := item.(*tview.InputField); ok {
			if field.GetLabel() == "FilePath" {
				filePath = field.GetText()
			}
		}
	}

	if filePath != "" {
		fileName = filepath.Base(filePath)
	}

	result := models.BinaryDataContent{
		FileName: fileName,
		Data:     fileData,
	}

	return result
}

func (t *TUI) processDataByType(form *tview.Form, req *models.DataRequest) {
	var data any

	switch req.Type {
	case models.LoginPassword:
		data = t.extractLoginPasswordData(form)
	case models.TextData:
		data = t.extractTextData(form)
	case models.CardData:
		data = t.extractCardData(form)
	case models.BinaryData:
		data = t.extractBinaryData(form)
	}

	t.encryptAndSaveData(data, req)
}

func (t *TUI) handleDecryptData(id int64) {
	currentData, dataIndex := t.findDataByID(id)
	if currentData == nil {
		t.showError("Данные не найдены")

		return
	}

	t.showPasswordDialog(func(password string) {
		t.decryptAndUpdateData(currentData, dataIndex, password)
	})
}

func (t *TUI) findDataByID(id int64) (*models.DataResponse, int) {
	for i, data := range t.dataList {
		if data.ID == id {
			return &t.dataList[i], i
		}
	}

	return nil, -1
}

func (t *TUI) decryptAndUpdateData(currentData *models.DataResponse, dataIndex int, password string) {
	encryptedData, err := t.client.GetEncryptedData(currentData.ID)
	if err != nil {
		t.showError(fmt.Sprintf("Ошибка получения зашифрованных данных: %v", err))

		return
	}

	decryptedContent, err := t.client.DecryptData(encryptedData, password)
	if err != nil {
		t.showError(fmt.Sprintf("Ошибка расшифровки данных: %v", err))

		return
	}

	if !t.validateDecryptedContent(currentData.Type, decryptedContent) {
		return
	}

	t.dataList[dataIndex].Content = decryptedContent
	currentData.Content = decryptedContent

	if t.setViewData != nil {
		t.setViewData(*currentData)
	}
}

func (t *TUI) validateDecryptedContent(dataType models.DataType, content any) bool {
	switch dataType {
	case models.BinaryData:
		if binaryData, ok := content.(models.BinaryDataContent); ok {
			if len(binaryData.Data) == 0 {
				t.showError("Предупреждение: расшифрованный файл не содержит данных")
			} else {
				t.showInfo(fmt.Sprintf("Файл успешно расшифрован: %s (%d байт)",
					binaryData.FileName, len(binaryData.Data)))
			}
		} else {
			t.showError("Ошибка: расшифрованные данные имеют неверный формат")

			return false
		}
	case models.TextData:
		if _, ok := content.(models.TextDataContent); !ok {
			t.showError("Ошибка: расшифрованные текстовые данные имеют неверный формат")

			return false
		}
		t.showInfo("Текст успешно расшифрован")
	case models.CardData:
		if _, ok := content.(models.CardDataContent); !ok {
			t.showError("Ошибка: расшифрованные данные карты имеют неверный формат")

			return false
		}
		t.showInfo("Данные карты успешно расшифрованы")
	case models.LoginPassword:
		if _, ok := content.(models.LoginPasswordData); !ok {
			t.showError("Ошибка: расшифрованные данные логина/пароля имеют неверный формат")

			return false
		}
		t.showInfo("Логин и пароль успешно расшифрованы")
	}

	return true
}

func (t *TUI) displayLoginPasswordContent(text *tview.TextView, content any) {
	if data, ok := content.(models.LoginPasswordData); ok {
		fmt.Fprintf(text, "[yellow]Логин:[white] %s\n", data.Login)
		fmt.Fprintf(text, "[yellow]Пароль:[white] %s\n", data.Password)
	} else {
		fmt.Fprintf(text, "Ошибка отображения данных логина/пароля\n")
	}
}

func (t *TUI) displayDataInTextView(text *tview.TextView, data models.DataResponse) {
	text.Clear()

	fmt.Fprintf(text, "[yellow]ID:[white] %d\n", data.ID)
	fmt.Fprintf(text, "[yellow]Тип:[white] %s\n", t.getDataTypeLabel(data.Type))
	fmt.Fprintf(text, "[yellow]Название:[white] %s\n", data.Name)
	fmt.Fprintf(text, "[yellow]Создано:[white] %s\n", data.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Fprintf(text, "[yellow]Обновлено:[white] %s\n\n", data.UpdatedAt.Format("2006-01-02 15:04"))

	if data.Content != nil {
		t.displayDataContentByType(text, data)
	} else {
		fmt.Fprintf(text, "[yellow]Данные зашифрованы. Нажмите 'Расшифровать' для просмотра.[white]\n")
	}
}

func (t *TUI) displayDataContentByType(text *tview.TextView, data models.DataResponse) {
	fmt.Fprintf(text, "[green]Содержимое:[white]\n")

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
		fmt.Fprintf(text, "Неизвестный тип данных: %s\n", data.Type)
	}
}

func (t *TUI) displayTextDataContent(text *tview.TextView, content any) {
	if data, ok := content.(models.TextDataContent); ok {
		fmt.Fprintf(text, "%s\n", data.Text)
	} else {
		fmt.Fprintf(text, "Ошибка отображения текстовых данных\n")
	}
}

func (t *TUI) displayCardDataContent(text *tview.TextView, content any) {
	if data, ok := content.(models.CardDataContent); ok {
		fmt.Fprintf(text, "[yellow]Номер карты:[white] %s\n", data.CardNumber)
		fmt.Fprintf(text, "[yellow]Имя владельца:[white] %s\n", data.CardHolder)
		fmt.Fprintf(text, "[yellow]Срок действия:[white] %s\n", data.ExpiryDate)
		fmt.Fprintf(text, "[yellow]CVV:[white] %s\n", data.CVV)
	} else {
		fmt.Fprintf(text, "Ошибка отображения данных карты\n")
	}
}

func (t *TUI) displayBinaryDataContent(text *tview.TextView, content any) {
	contentType := fmt.Sprintf("%T", content)

	if data, ok := content.(models.BinaryDataContent); ok {
		fmt.Fprintf(text, "[yellow]Имя файла:[white] %s\n", data.FileName)
		fmt.Fprintf(text, "[yellow]Размер:[white] %d байт\n", len(data.Data))
	} else {
		fmt.Fprintf(text, "Ошибка отображения бинарных данных\n")
		fmt.Fprintf(text, "Тип полученных данных: %s\n", contentType)
	}
}

func (t *TUI) extractIDFromText(text *tview.TextView) int64 {
	content := text.GetText(true)

	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "[yellow]ID:[white]") {
			idStr := strings.TrimSpace(strings.TrimPrefix(line, "[yellow]ID:[white]"))
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err == nil {
				return id
			}
		} else if strings.HasPrefix(line, "ID:") {
			idStr := strings.TrimSpace(strings.TrimPrefix(line, "ID:"))
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err == nil {
				return id
			}
		}
	}

	return 0
}

func (t *TUI) handleDeleteData(id int64) {
	t.showDialog(
		"Подтверждение удаления",
		fmt.Sprintf("Вы уверены, что хотите удалить запись с ID %d?", id),
		"Удалить",
		func() {
			err := t.client.DeleteData(context.Background(), id)
			if err != nil {
				t.showError(fmt.Sprintf("Ошибка удаления: %v", err))

				return
			}

			t.showInfo("Запись успешно удалена")
			t.loadData()
			t.pages.SwitchToPage("main")
		},
	)
}

func (t *TUI) loadData() {
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

func (t *TUI) saveFile(fileName string, data []byte) {
	t.showFileDialogForDir(func(dirPath string) {
		filePath := filepath.Join(dirPath, fileName)

		if _, err := os.Stat(filePath); err == nil {
			t.showDialog(
				"Подтверждение перезаписи",
				fmt.Sprintf("Файл %s уже существует. Перезаписать?", fileName),
				"Перезаписать",
				func() {
					t.writeFile(filePath, data)
				},
			)
		} else {
			t.writeFile(filePath, data)
		}
	})
}

func (t *TUI) writeFile(filePath string, data []byte) {
	err := os.WriteFile(filePath, data, 0600)
	if err != nil {
		t.showError(fmt.Sprintf("Ошибка сохранения файла: %v", err))
	} else {
		t.showInfo(fmt.Sprintf("Файл успешно сохранен: %s", filePath))
	}
}
