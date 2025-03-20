<h1> Настройка перед запуском </h1> <br>
<br>
<h5> Для запуска необходимо скачать GO - <a href='https://go.dev/dl/'>Ссылка</a> </h5> <br>
<br>
<h5> Также необходим файл .env с входящими него нужными для входа сервисы данными</h5> <br>
<br>
<h5> Перед запуском нужно написать команду <u> go mod tidy </u>, для синхронизации данных библиотек </h5> <br>
<br>
<h1> Работа </h1> <br>
<br>
<h3> Перед работой с программой неодимо задать имена файлов, который требуется искать и где конкретно </h3>
<ul> 
    <h5> Для парсинга из s-bki </h5>
    <li> Задаем список файлов, который нужно искать, переменная - filenames2</li>
    <li> Раскоменчиваем список форматов, переменная - formats </li>
    <li> Задаем папки, в которых искать - Меняем просто Бюро, которого нужно найти </li>
    <li> Задаем папку, в которую нужно сохранять, по умолчанию - C:\output</li>
    <br>
    <h5> Для парсинга из почты </h5>
    <li> Задаем файлы, которые нужно найти, переменная - fileList </li>
    <li> Может работать некорректно - todo </li>
    <br>
    <h5> Для парсинга лк НБКИ </h5>
    <li> Задаем файлы, которые нужно найти, переменная - fileList</li>
</ul>
<h3> После запуска через <u> go run main.go </u> программа выдаст ошибку, нужно ввести флаги </h3>
<ul>
    <h5> В зависимости от флага, программа начнет парсинг, флаги можно комбинировать</h5>
    <li> go run main.go -parse </li>
    <li> go run main.go -selenium </li>
    <li> go run main.go -email </li>
</ul>

