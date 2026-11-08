# Mirai - план по IPA и APK (без Expo, сборка на GitHub Actions)

Сейчас делаем только ПК + сервис. Мобилки после, этот план чтобы не переделывать контракт.

## Принцип
Мобильные клиенты не знают про AniLiberty/Yummy, ходят в тот же Go-сервер по `docs/API.md`. Значит все "фиксы на аниме" (склейка, озвучки, дубли) живут в одном месте.

## Сервер для телефонов
На телефоне локального сервера нет, поэтому тот же Go-бинарник выкатываем на VPS (Frankfurt-3, `mirai.savsis.xyz`, nginx + LE, systemd). Один обход каталога на всех, телефоны только читают из памяти. Домашний провайдер режет часть финских IP, Frankfurt-3 с домашней сети доступен (проверено на anisav.savsis.xyz). Десктоп может переключиться на этот же сервер в настройках, если локальный не нужен.
Позже, если захочется работать без сервера: `gomobile bind` собирает тот же Go-движок в AAR и xcframework (минус: каждый телефон сам обходит 200+ запросов).

## Android (APK)
- Нативно: Kotlin + Jetpack Compose, minSdk 25 (Android 7.1), target/compile - последний. Без Expo/React Native.
- Видео: Media3 ExoPlayer (HLS AniLibria, выбор 480/720/1080, пропуск опенинга), embed-плееры через WebView, загружающий `/v1/embed`.
- Сеть: OkHttp + kotlinx.serialization, картинки через Coil (кэш на диске), Compose lazy-сетки.
- Для 7.1 нужен desugaring и cleartext off (сервер по https), TLS-сертификаты LE на 7.1 живут (ISRG Root X1 через сеть/пакет).
- CI: `.github/workflows/android.yml`, ubuntu-latest, `actions/setup-java` 17, `gradle assembleRelease`, подпись keystore из GitHub Secrets (сделаем свой, не debug), артефакт `Mirai.apk` и GitHub Release по тегу.

## iOS (IPA)
- Нативно: Swift + SwiftUI, deployment target iOS 18.0, проверка на 18 ... 27 (симуляторы в CI матрицей).
- Видео: AVPlayer + AVKit (HLS из коробки, PiP, AirPlay), embed-плееры через WKWebView `loadHTMLString(baseURL: https://mirai.savsis.xyz)` чтобы был нормальный Referer.
- Сеть: URLSession + Codable, картинки через свой кэш на URLCache/NSCache.
- CI: `.github/workflows/ios.yml`, macos-latest, `xcodebuild archive` без подписи (CODE_SIGNING_ALLOWED=NO), сборка `.ipa` вручную упаковкой Payload, артефакт `Mirai.ipa` для сайдлоада бесплатным Apple ID (резайн раз в 7 дней, как с AniSav).
- Мелкие требования Apple: ATS (https), фоновое аудио только если нужен.

## Порядок
1. Закончить ПК + сервис, выложить сервер на Frankfurt-3.
2. Android: каталог + тайтл + HLS плеер, потом embed, потом избранное/история (APK быстрее собирается и тестируется).
3. iOS по тому же списку экранов.
4. Один дизайн-язык (тушь/бумага/киноварь, манга-панели), токены в `docs/design-tokens.md` для переноса.
5. Общая проверка: один и тот же `/v1/*` контракт, версия API в `/v1/status`.

## Риски
- Yummy "для личного использования": публичный сервер должен быть без рекламы авторства и с мягким rate-limit.
- Embed-плееры (Kodik/Alloha) капризны к Referer и WebView - первые баги будут тут.
- Windows 10, Android 7.1, iOS 18 - долгий хвост совместимости, на старых версиях честно предупреждаем.
