## В данном репозитории приведён вариант решения задачи l2.7  

### ⚙️ Условие задачи:  

Что выведет программа?
Объяснить работу конвейера с использованием select.

    package main

    import (
	    "fmt"
	    "math/rand"
	    "time"
    )

    func asChan(vs ...int) <-chan int {
	    c := make(chan int)
	    go func() {
		    for _, v := range vs {
			    c <- v
			    time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
		    }
		    close(c)
	    }()
	    return c
    }

    func merge(a, b <-chan int) <-chan int {
	    c := make(chan int)
	    go func() {
		    for {
			    select {
			    case v, ok := <-a:
				    if ok {
					    c <- v
				    } else {
					    a = nil
				    }
			    case v, ok := <-b:
				    if ok {
					    c <- v
				    } else {
					    b = nil
				    }
			    }
			    if a == nil && b == nil {
				    close(c)
				    return
			    }
		    }
	    }()
	    return c
    }

    func main() {

	    rand.Seed(time.Now().Unix())
	    a := asChan(1, 3, 5, 7)
	    b := asChan(2, 4, 6, 8)
	    c := merge(a, b)
	    for v := range c {
		    fmt.Print(v)
	    }
    }
    
### 📋 Перечень решений:

- main.go - решение задачи l2.7  
