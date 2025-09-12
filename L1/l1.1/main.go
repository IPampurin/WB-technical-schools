/*   История острова сокровищ в трёх структурах и трёх методах   */

package main

import (
	"fmt"
)

// Допустим, имеется структура Human

// Human - характеристики из досье
type Human struct {
	Profession    string // вид занятости
	Style         string // что за человек
	Name          string // имя
	Hobby         string // что любит
	Character     string // характер
	Age           int    // возраст
	MaritalStatus bool   // женат / не женат
}

// У типа Human есть следующие методы

// Present выводит информацию в виде справки
func (h *Human) Present() {

	message := fmt.Sprintf("%s, %s, возраст - %d лет, %s человек, любит %s, характер %s, %s.",
		h.Name,
		h.Profession,
		h.Age,
		h.Style,
		h.Hobby,
		h.Character,
		func() string {
			if h.MaritalStatus {
				return "женат"
			}
			return "не женат"
		}())

	fmt.Println(message)
}

// AgeChange добавляет персонажу возраст в размере срока путешествия
func (h *Human) AgeChange(yearsInVoyage int) {

	h.Age += yearsInVoyage
}

// Для динамики добавим структуру Changes с изменениями после событий

// Changes описывает изменения персонажей
type Changes struct {
	ChangProf      string // новый род занятий
	ChangStyle     string // новый стиль
	ChangHobby     string // новое любимое занятие
	ChangCharacter string // каким стал характер
}

// Также есть структура Action

// Action включает характеристики после известных событий
type Action struct {
	Human
	Changes
	Married bool // женился / не женился
}

// У типа Action есть следующий метод

// InitChanges применяет изменения к вложенному типу в зависимости от семейного положения
func (a *Action) InitChanges() {

	a.Human.Profession = a.Changes.ChangProf
	a.Human.Style = a.Changes.ChangStyle
	a.Human.Hobby = a.Changes.ChangHobby
	a.Human.Character = a.Changes.ChangCharacter

	if a.Married {
		a.Human.MaritalStatus = true
	} else {
		a.Human.MaritalStatus = false
	}
}

func main() {

	// зададим начальные характеристики команды
	team := beginValues()

	fmt.Println()
	fmt.Printf("До поиска сокровищ:\n\n")

	// применяем метод Present и смотрим начальные досье (метод для Human работает для Action)
	for _, v := range team {
		v.Present()
	}

	fmt.Println()

	// задаём время путешествия и применяем метод AgeChange типа Human к персонажам по списку (метод для Human работает для Action)
	timeVoyage := 2
	for _, v := range team {
		v.AgeChange(timeVoyage)
	}

	// вносим изменения методом InitChanges (метод для Action)
	for _, v := range team {
		v.InitChanges()
	}

	fmt.Printf("После путешествия:\n\n")

	// с помощью метода Present (метод для Human) выводим досье чем всё закончилось
	for _, v := range team {
		v.Present()
	}
}

// beginValues служит для заполнения начальных условий
func beginValues() []*Action {

	team := []*Action{
		{
			Human: Human{
				Profession:    "юнга",
				Style:         "очень, очень хороший",
				Name:          "Джимми Хокинс",
				Age:           14,
				Hobby:         "слушать маму и делать по утрам зарядку",
				Character:     "очень мягкий",
				MaritalStatus: false,
			},
			Married: true,
			Changes: Changes{
				ChangProf:      "эсквайр",
				ChangStyle:     "очень, очень неосторожный",
				ChangHobby:     "виски и азартные игры",
				ChangCharacter: "отсутствует",
			},
		},
		{
			Human: Human{
				Profession:    "капитан",
				Style:         "военный",
				Name:          "Смоллет",
				Age:           50,
				Hobby:         "говорить правду в глаза, от чего и страдает",
				Character:     "прескверный",
				MaritalStatus: false,
			},
			Married: true,
			Changes: Changes{
				ChangProf:      "адмирал",
				ChangStyle:     "довольный жизнью",
				ChangHobby:     "говорить правду в глаза, но уже от этого не страдает",
				ChangCharacter: "просто скверный",
			},
		},
		{
			Human: Human{
				Profession:    "доктор",
				Style:         "очень хороший и весёлый",
				Name:          "Ливси",
				Age:           45,
				Hobby:         "улыбаться и давать советы о здоровье",
				Character:     "общительный",
				MaritalStatus: false,
			},
			Married: false,
			Changes: Changes{
				ChangProf:      "доктор",
				ChangStyle:     "довольный жизнью и весёлый",
				ChangHobby:     "улыбаться и играть в карты",
				ChangCharacter: "общительный",
			},
		},
		{
			Human: Human{
				Profession:    "сквайр",
				Style:         "жадный, трусливый и надменный",
				Name:          "Трелони",
				Age:           55,
				Hobby:         "есть и лениться",
				Character:     "отсутствует",
				MaritalStatus: true,
			},
			Married: true,
			Changes: Changes{
				ChangProf:      "сквайр",
				ChangStyle:     "трусливый и угрюмый",
				ChangHobby:     "виски и прокрастинировать",
				ChangCharacter: "отсутствует",
			},
		},
		{
			Human: Human{
				Profession:    "самый страшный пират",
				Style:         "очень опасный",
				Name:          "Джон Сильвер, он же 'Окорок', он же 'Одноногий'",
				Age:           55,
				Hobby:         "деньги",
				Character:     "скрытный",
				MaritalStatus: false,
			},
			Married: false,
			Changes: Changes{
				ChangProf:      "бывший социопат",
				ChangStyle:     "довольный жизнью и спокойный",
				ChangHobby:     "рыбалку и смотреть на закат",
				ChangCharacter: "самодостаточный",
			},
		},
	}

	return team
}
