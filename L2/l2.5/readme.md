## В данном репозитории приведён вариант решения задачи l2.5  

### ⚙️ Условие задачи:  

Что выведет программа?
Объяснить вывод программы.

    package main

    type customError struct {
      msg string
    }

    func (e *customError) Error() string {
      return e.msg
    }

    func test() *customError {
      // ... do something
      return nil
    }

    func main() {
      var err error
      err = test()
      if err != nil {
        println("error")
        return
      }
      println("ok")
    }
    
### 📋 Перечень решений:

- main.go - решение задачи l2.5  
