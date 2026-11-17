# Chrome Web Store

**Language:** Russian  
**Category:** Workflow and planning, if available  
**Support:** https://github.com/piksel-man231/wt-live-skin-installer/issues  
**Privacy policy:** https://github.com/piksel-man231/wt-live-skin-installer/blob/main/store/PRIVACY.md

## Описание

Устанавливайте пользовательские камуфляжи War Thunder прямо со страницы WT Live. Расширение добавляет кнопку «Установить скин» рядом со скачиванием и раскладывает файлы по папкам в UserSkins.

Наборы для нескольких машин устанавливаются за один раз. Папки получают названия из скачанного архива. Если папка с таким именем уже есть, следующая получит номер в скобках.

Для папки игры в AppData на Windows установите помощник один раз. Он доступен в разделе Releases проекта на GitHub. После настройки результат установки появится прямо на кнопке WT Live.

Независимый проект сообщества. Разработчики War Thunder и WT Live не поддерживают это расширение.

## Privacy practices

**Single purpose:** Install user selected WT Live camouflages into War Thunder's UserSkins folder.

**Native messaging:** On Windows, access to the game directory under AppData requires the separately installed local helper. The extension sends the selected WT Live post ID to the helper. The helper downloads the corresponding camouflage archive from WT Live and saves its contents in the configured UserSkins directory.

**Host permissions:** live.warthunder.com is used to add the install button and retrieve the selected camouflage's metadata. cdn-live.warthunder.com serves the archive.

**Remote code:** No. The extension code and ZIP library are packaged locally. Downloaded archives are installed as skin data.

## Проверка для модератора

Откройте https://live.warthunder.com/post/1173999/en/ и нажмите «Установить скин». Без помощника расширение предложит выбрать папку UserSkins. Для проверки распаковки можно создать пустую папку с этим именем в «Документах». Выберите её и нажмите «Установить». Этот архив содержит восемь скинов.

## Сборка после получения ID магазина

Загрузите ZIP с содержимым папки `extension` через Developer Dashboard. Затем из папки `native` соберите Windows-установщик с фактическим ID карточки:

```sh
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
  -ldflags="-s -w -H windowsgui -X main.storeExtensionID=STORE_ITEM_ID" \
  -o WT-Live-Setup.exe ./setup
```

Опубликуйте эту сборку в GitHub Releases после того, как карточка станет доступна посетителям. При переходе с распакованной версии отключите её после установки версии из магазина, чтобы на WT Live работала одна копия расширения.
