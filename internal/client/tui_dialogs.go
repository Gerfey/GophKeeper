package client

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rivo/tview"
)

func (t *TUI) showError(message string) {
	t.showDialog("Ошибка", message, "OK", nil)
}

func (t *TUI) showInfo(message string) {
	t.showDialog("Информация", message, "OK", nil)
}

func (t *TUI) showDialog(title, message, buttonText string, callback func()) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{buttonText}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonIndex == 0 && callback != nil {
				callback()
			}
			t.pages.RemovePage("dialog")
		})

	if title != "" {
		modal.SetTitle(title).SetBorder(true)
	}

	t.pages.AddPage("dialog", modal, true, true)
}

func (t *TUI) showCreateMasterPasswordModal() {
	form := tview.NewForm()

	passwordField := tview.NewInputField().
		SetLabel("Мастер-пароль").
		SetFieldWidth(passwordFieldWidth).
		SetMaskCharacter('*')

	confirmPasswordField := tview.NewInputField().
		SetLabel("Подтверждение пароля").
		SetFieldWidth(passwordFieldWidth).
		SetMaskCharacter('*')

	form.AddFormItem(passwordField)
	form.AddFormItem(confirmPasswordField)

	form.AddButton("Сохранить", func() {
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

		err := t.client.SetMasterPassword(password)
		if err != nil {
			t.showError(fmt.Sprintf("Ошибка установки мастер-пароля: %v", err))
			return
		}

		t.config.HasMasterPassword = true
		if err := t.saveConfig(); err != nil {
			t.showError(fmt.Sprintf("Ошибка сохранения конфигурации: %v", err))
		}

		t.pages.RemovePage("masterPasswordModal")
		t.pages.SwitchToPage("main")
	})

	form.AddButton("Отмена", func() {
		t.pages.RemovePage("masterPasswordModal")
		t.pages.SwitchToPage("main")
	})

	form.SetTitle("Создание мастер-пароля").
		SetBorder(true).
		SetTitleAlign(tview.AlignCenter)

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, dialogWidth, 1, true).
			AddItem(nil, 0, 1, false),
			dialogHeight, 1, true).
		AddItem(nil, 0, 1, false)

	t.pages.AddPage("masterPasswordModal", flex, true, true)
}

func (t *TUI) showPasswordDialogFixed(callback func(password string)) {
	form := tview.NewForm()

	passwordField := tview.NewInputField().
		SetLabel("Мастер-пароль").
		SetFieldWidth(passwordFieldWidth).
		SetMaskCharacter('*')

	form.AddFormItem(passwordField)

	form.AddButton("OK", func() {
		password := passwordField.GetText()
		if password == "" {
			t.showError("Пароль не может быть пустым")
			return
		}

		t.pages.RemovePage("passwordDialog")
		if callback != nil {
			callback(password)
		}
	})

	form.AddButton("Отмена", func() {
		t.pages.RemovePage("passwordDialog")
	})

	form.SetTitle("Введите мастер-пароль").
		SetBorder(true).
		SetTitleAlign(tview.AlignCenter)

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, dialogWidth, 1, true).
			AddItem(nil, 0, 1, false),
			dialogHeight, 1, true).
		AddItem(nil, 0, 1, false)

	t.pages.AddPage("passwordDialog", flex, true, true)
	t.app.SetFocus(passwordField)
}

func (t *TUI) showPasswordDialog(callback func(password string)) {
	t.showPasswordDialogFixed(callback)
}

func (t *TUI) showFileDialog(callback func(filePath string)) {
	var currentDir string

	if t.lastFileDialogDir != "" {
		currentDir = t.lastFileDialogDir
	} else {
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
			SetFieldWidth(50)

		form := tview.NewForm()
		form.AddButton("Выбрать", func() {
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
			AddItem(form, 3, 0, false)

		modal := tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(nil, 0, 1, false).
				AddItem(layout, 60, 1, true).
				AddItem(nil, 0, 1, false), 20, 1, true).
			AddItem(nil, 0, 1, false)

		modal.SetBorder(true).
			SetTitle("Выберите файл").
			SetTitleAlign(tview.AlignCenter)

		t.pages.AddPage("file_dialog", modal, true, true)
		t.app.SetFocus(list)
	}

	updateFileList(currentDir)
}

func (t *TUI) showFileDialogForDir(callback func(dirPath string)) {
	var currentDir string

	if t.lastFileDialogDir != "" {
		currentDir = t.lastFileDialogDir
	} else {
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
			t.pages.RemovePage("file_dialog_dir")
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
					t.pages.RemovePage("file_dialog_dir")
					updateFileList(path)
				})
			}
		}

		inputField := tview.NewInputField().
			SetLabel("Путь: ").
			SetFieldWidth(50).
			SetText(dir)

		form := tview.NewForm()
		form.AddButton("Выбрать эту директорию", func() {
			t.pages.RemovePage("file_dialog_dir")
			callback(dir)
		})
		form.AddButton("Создать директорию", func() {
			newDirName := inputField.GetText()
			if newDirName == "" {
				t.showError("Введите имя директории")
				return
			}

			if !filepath.IsAbs(newDirName) {
				newDirName = filepath.Join(dir, newDirName)
			}

			if _, err := os.Stat(newDirName); err == nil {
				t.showError("Директория уже существует")
				return
			}

			if err := os.MkdirAll(newDirName, 0755); err != nil {
				t.showError(fmt.Sprintf("Ошибка создания директории: %v", err))
				return
			}

			t.pages.RemovePage("file_dialog_dir")
			updateFileList(newDirName)
		})
		form.AddButton("Отмена", func() {
			t.pages.RemovePage("file_dialog_dir")
		})
		form.SetButtonsAlign(tview.AlignCenter)

		layout := tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(tview.NewTextView().SetText("Текущая директория: "+dir), 1, 0, false).
			AddItem(list, 0, 1, true).
			AddItem(inputField, 1, 0, false).
			AddItem(form, 3, 0, false)

		modal := tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(nil, 0, 1, false).
				AddItem(layout, 60, 1, true).
				AddItem(nil, 0, 1, false), 20, 1, true).
			AddItem(nil, 0, 1, false)

		modal.SetBorder(true).
			SetTitle("Выберите директорию").
			SetTitleAlign(tview.AlignCenter)

		t.pages.AddPage("file_dialog_dir", modal, true, true)
		t.app.SetFocus(list)
	}

	updateFileList(currentDir)
}
