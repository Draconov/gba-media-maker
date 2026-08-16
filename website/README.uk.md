<div align="center">
  <img src="../assets/icon.png" width="88" alt="Іконка GBA Media Maker">

# GBA Media Maker Web

**Браузерна версія конвертера GBA Media Maker. Обробка виконується локально на вашому пристрої.**

[English](README.md) · **Українська**
</div>

## Запуск

Для звичайного користування відкрийте розгорнуту вебверсію. Для локальної розробки:

```bash
npm install
npm run dev
```

Production build:

```bash
npm run build
```

Тести:

```bash
npm test
```

## Мови

Інтерфейс підтримує **English** і **Українська**.

- на першому запуску використовується мова браузера/системи;
- вибір користувача зберігається у `localStorage`;
- англійська використовується як fallback;
- GBA ROM/player UI не локалізується цим механізмом.

Єдине джерело локалізації знаходиться у корені репозиторію:

```text
../locales/index.json
../locales/en.json
../locales/uk.json
```

`../locales/index.json` керує списком мов у dropdown-меню. Щоб додати нову мову пізніше, додайте каталог і один запис у manifest. Перед `dev`, `test` і `build` скрипт `scripts/sync-locales.mjs` копіює всі locale JSON у згенеровану директорію `public/locales/`. Не редагуйте згенеровану копію вручну.

## Що підтримує вебверсія

- відео, GIF, аудіо та статичні зображення;
- одиночні ROM, медіаменю та ZIP з окремими ROM;
- обрізання, швидкість, fit/crop/stretch;
- palette/dithering/compression налаштування;
- PCM і експериментальний ADPCM;
- аудіообкладинки;
- власні фони меню;
- титульні картки;
- Extreme/Smart optimization;
- довге відео з автоматичним splitting;
- `.gbamedia` проєкти з повторною прив’язкою вихідних файлів.

## Відмінність від Windows-застосунку

Формати й параметри конвертації тримаються у parity з десктопною версією, але браузер має жорсткіші обмеження пам’яті та доступу до файлової системи. Для дуже великих відео краще використовувати Windows-застосунок з нативним FFmpeg.

## Приватність

Вихідні файли не надсилаються на сервер для конвертації. `ffmpeg.wasm` та ROM builder працюють локально у браузері.

## Структура важливих файлів

```text
src/main.js              головний UI/конвертація
src/i18n.js              runtime локалізації
../locales/*.json        канонічні переклади
src/rom-core.js          ROM builder
src/smart-encoding.js    Extreme/Smart аналіз
src/menu-themes.js       теми меню
src/title-cards.js       титульні картки
scripts/sync-locales.mjs синхронізація локалей
```
