## В данном репозитории приведён вариант решения задачи l2.6  

### ⚙️ Условие задачи:  

Что выведет программа?

Объяснить поведение срезов при передаче их в функцию.

    package main

    import (
      "fmt"
    )

    func main() {
      var s = []string{"1", "2", "3"}
      modifySlice(s)
      fmt.Println(s)
    }

    func modifySlice(i []string) {
      i[0] = "3"
      i = append(i, "4")
      i[1] = "5"
      i = append(i, "6")
    }
    
### 📋 Перечень решений:

- main.go - решение задачи l2.6  
