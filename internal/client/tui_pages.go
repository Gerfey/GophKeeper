package client

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/gerfey/gophkeeper/internal/models"
)

const (
	buttonWidth      = 40
	buttonAreaHeight = 3
)

func (t *TUI) createMainPage() tview.Primitive {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	table := tview.NewTable().
		SetBorders(true).
		SetSelectable(true, false)

	table.SetCell(0, idColumn, tview.NewTableCell("ID").SetTextColor(tcell.ColorYellow).SetSelectable(false))
	table.SetCell(0, typeColumn, tview.NewTableCell("Тип").SetTextColor(tcell.ColorYellow).SetSelectable(false))
	table.SetCell(0, nameColumn, tview.NewTableCell("Название").SetTextColor(tcell.ColorYellow).SetSelectable(false))
	table.SetCell(
		0,
		updatedColumn,
		tview.NewTableCell("Обновлено").SetTextColor(tcell.ColorYellow).SetSelectable(false),
	)

	t.dataTable = table

	t.updateTable = func() {
		t.updateDataTable(t.dataList)
	}

	createViewPageForData := func(data models.DataResponse) {
		t.createViewPageForData(data)
	}

	table.SetSelectedFunc(func(row, _ int) {
		if row > 0 && row <= len(t.dataList) {
			data := t.dataList[row-1]
			createViewPageForData(data)
		}
	})

	form := tview.NewForm()
	form.AddButton("Добавить", func() {
		t.pages.SwitchToPage("add")
	})
	form.AddButton("Просмотр", func() {
		if row, _ := table.GetSelection(); row > 0 && row <= len(t.dataList) {
			data := t.dataList[row-1]
			createViewPageForData(data)
		}
	})
	form.AddButton("Выход", func() {
		t.logout()
	})

	buttonsLayout := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(form, buttonWidth, 1, true).
		AddItem(nil, 0, 1, false)

	flex.AddItem(table, 0, 1, true).
		AddItem(buttonsLayout, buttonAreaHeight, 0, false)

	flex.SetTitle("GophKeeper - Данные").SetBorder(true)

	return flex
}

func (t *TUI) createAddPage() tview.Primitive {
	form := tview.NewForm()
	form.SetBorder(true).SetTitle("Добавить данные")

	dataTypes := []string{"Логин/Пароль", "Текст", "Карта", "Файл"}
	currentTypeIndex := 0

	updateFormFields := func(typeIndex int) {
		dropdown := form.GetFormItem(0)

		form.Clear(true)

		form.AddFormItem(dropdown)

		nameField := t.addNameField(form)
		t.addDataTypeSpecificFields(form, dataTypes[typeIndex])
		t.addAddPageButtons(form, nameField, dataTypes, typeIndex)
	}

	form.AddDropDown("Тип данных", dataTypes, currentTypeIndex, func(_ string, index int) {
		if index != currentTypeIndex {
			currentTypeIndex = index
			updateFormFields(index)
		}
	})

	nameField := t.addNameField(form)
	t.addDataTypeSpecificFields(form, dataTypes[currentTypeIndex])
	t.addAddPageButtons(form, nameField, dataTypes, currentTypeIndex)

	return form
}

func (t *TUI) createViewPageButtons(text *tview.TextView, data models.DataResponse) *tview.Form {
	buttons := tview.NewForm()

	buttons.AddButton("Расшифровать", func() {
		id := t.extractIDFromText(text)
		if id > 0 {
			t.handleDecryptData(id)
		} else {
			t.showError("Не удалось получить ID записи")
		}
	})

	if data.Type == models.BinaryData {
		buttons.AddButton("Скачать", func() {
			if data.Content == nil {
				t.showError("Сначала необходимо расшифровать данные")

				return
			}

			binaryData, ok := data.Content.(models.BinaryDataContent)
			if !ok {
				t.showError("Ошибка: неверный формат данных")

				return
			}

			if len(binaryData.Data) == 0 {
				t.showError("Ошибка: файл пуст или данные не были корректно расшифрованы")

				return
			}

			t.saveFile(binaryData.FileName, binaryData.Data)
		})
	}

	buttons.AddButton("Удалить", func() {
		id := t.extractIDFromText(text)
		if id > 0 {
			t.handleDeleteData(id)
		} else {
			t.showError("Не удалось получить ID записи")
		}
	})

	buttons.AddButton("Назад", func() {
		t.pages.SwitchToPage("main")
	})

	return buttons
}

func (t *TUI) createViewPageForData(data models.DataResponse) {
	text := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)

	t.setViewData = func(updatedData models.DataResponse) {
		text.Clear()
		t.displayDataInTextView(text, updatedData)
	}

	t.displayDataInTextView(text, data)

	buttons := t.createViewPageButtons(text, data)

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(text, 0, 1, false).
		AddItem(buttons, buttonAreaHeight, 0, true)

	t.pages.AddPage("view", flex, true, true)
}

func (t *TUI) createViewPage() tview.Primitive {
	text := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWrap(true).
		SetTextAlign(tview.AlignLeft)

	buttons := t.createViewPageButtons(text, models.DataResponse{})

	buttonsLayout := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(buttons, buttonWidth, 1, true).
		AddItem(nil, 0, 1, false)

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(text, 0, 1, false).
		AddItem(buttonsLayout, buttonAreaHeight, 0, true)

	flex.SetTitle("GophKeeper - Просмотр данных").SetBorder(true)

	return flex
}
