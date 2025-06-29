package client

import (
	"context"
	"fmt"

	"github.com/rivo/tview"
)

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

	t.showDialog(
		"Успешная регистрация",
		fmt.Sprintf("Пользователь %s успешно зарегистрирован", username),
		"OK",
		func() {
			t.pages.SwitchToPage("login")
		},
	)
}

func (t *TUI) addRegisterFormNavigationButtons(form *tview.Form) {
	form.AddButton("Назад", func() {
		t.pages.SwitchToPage("login")
	})

	form.SetTitle("GophKeeper - Регистрация").SetBorder(true)
}
