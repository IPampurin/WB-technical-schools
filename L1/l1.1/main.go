/*   История острова сокровищ в трёх структурах и четырёх методах   */

package main

import "fmt"

// Допустим, имеется структура Human

// Human - характеристи из досье
type Human struct {
	Profession    string // вид занятости
	Style         string // что за человек
	Name          string // имя
	Age           int    // возраст
	Hobby         string // что любит
	Character     string // характер
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

// Voyage добавляет персонажу возраст в размере срока путешествия
func (h *Human) Voyage(duration int) {

	h.Age += duration
}

// Для динамики добавим структуру Changes с изменениями

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
	Married bool // женился / не женился
	Rich    bool // разбогател / не разбогател
	Changes
}

// У типа Action есть следующие методы

// func (a *Action)

func main() {

	// зададим начальные характеристики команды
	team := []*Action{
		{Human: Human{
			Profession:    "юнга",
			Style:         "очень, очень хороший",
			Name:          "Джимми Хокинс",
			Age:           14,
			Hobby:         "слушать маму и делать по утрам зарядку",
			Character:     "очень мягкий",
			MaritalStatus: false,
		},
			Married: true,
			Rich:    true,
			Changes: Changes{
				ChangProf:      "эсквайр",
				ChangStyle:     "очень, очень неосторожный",
				ChangHobby:     "прокрастинировать",
				ChangCharacter: "отсутствует",
			},
		},
		{Human: Human{
			Profession:    "капитан",
			Style:         "военный",
			Name:          "Смоллет",
			Age:           50,
			Hobby:         "говорить правду в глаза, от чего и страдает",
			Character:     "прескверный",
			MaritalStatus: false,
		},
			Married: false,
			Rich:    true,
			Changes: Changes{
				ChangProf:      "адмирал",
				ChangStyle:     "довольный жизнью",
				ChangHobby:     "говорить правду в глаза, но уже от этого не страдает",
				ChangCharacter: "просто скверный",
			},
		},
		{Human: Human{
			Profession:    "доктор",
			Style:         "очень хороший и весёлый",
			Name:          "Ливси",
			Age:           45,
			Hobby:         "улыбаться и давать советы о здоровье",
			Character:     "общительный",
			MaritalStatus: false,
		},
			Married: false,
			Rich:    true,
			Changes: Changes{
				ChangProf:      "доктор",
				ChangStyle:     "довольный жизнью и весёлый",
				ChangHobby:     "улыбаться и играть в карты",
				ChangCharacter: "общительный",
			},
		},
		{Human: Human{
			Profession:    "сквайр",
			Style:         "жадный, трусливый и надменный",
			Name:          "Трелони",
			Age:           55,
			Hobby:         "есть и лениться",
			Character:     "отсутствует",
			MaritalStatus: true,
		},
		},
		{Human: Human{
			Profession:    "самый страшный пират",
			Style:         "очень опасный",
			Name:          "Джон Сильвер, он же 'Окорок', он же 'Одноногий'",
			Age:           55,
			Hobby:         "деньги",
			Character:     "скрытный",
			MaritalStatus: false,
		},
		},
	}

	fmt.Println("До путешествия:")

	// применяем метод Present и смотрим начальные досье
	for _, v := range team {
		v.Present()
	}

	fmt.Println()

	// применяем метод Voyage типа Human к персонажам по списку
	for _, v := range team {
		v.Voyage(2)
	}

	fmt.Println("После путешествия:")

	// с помощью метода Present выводим досье чем всё закончилось
	for _, v := range team {
		v.Present()
	}
}
