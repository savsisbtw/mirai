package main

import "strings"

var genreAliases = map[string]string{}

var typeAliases = map[string]string{}

func init() {
	genrePairs := [][2]string{
		{"Сёнен", "Сёнэн"},
		{"Сейнен", "Сэйнэн"},
		{"Дзёсей", "Дзёсэй"},
		{"Исекай", "Исэкай"},
		{"Сёдзе", "Сёдзё"},
		{"Сёдзе-ай", "Сёдзё-ай"},
		{"Психологическое", "Психология"},
		{"Супер сила", "Суперспособности"},
		{"Школа", "Школьная жизнь"},
	}
	for _, pair := range genrePairs {
		genreAliases[norm(pair[0])] = pair[1]
	}
	typePairs := [][2]string{
		{"ОNA", "ONA"},
		{"OAD", "OVA"},
		{"п/ф", "Фильм"},
		{"к/ф", "Короткометражка"},
		{"м/с", "Мини-сериал"},
		{"спешл", "Спешл"},
		{"Неизвестно", ""},
	}
	for _, pair := range typePairs {
		typeAliases[norm(pair[0])] = pair[1]
	}
}

func canonGenre(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if canon, ok := genreAliases[norm(name)]; ok {
		return canon
	}
	return name
}

func canonType(name string) string {
	name = strings.TrimSpace(name)
	if canon, ok := typeAliases[norm(name)]; ok {
		return canon
	}
	return name
}
