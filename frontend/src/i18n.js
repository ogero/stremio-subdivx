import i18n from "i18next";
import {initReactI18next} from "react-i18next";
import LanguageDetector from 'i18next-browser-languagedetector';

const resources = {
    en: {
        translation: {
            "The definitive Spanish subtitles": "The definitive Spanish subtitles",
            "Never Miss": "Never Miss",
            "A Single Line": "A Single Line",
            "Description": "Access the ultimate library of Spanish subtitles from Subdivx right inside Stremio. Enjoy your favorite movies and series with perfect timing.",
            "Install Now": "Install Now",
            "Buy Me a Coffee": "Buy Me a Coffee on cafecito.app",
            "Donate": "Donate",
            "Install manually": "Or manually install it by copying the manifest URL:",
            "Disclaimer": "We are not affiliated, associated, authorized, endorsed by, or in any way officially connected with Subdivx.com.",
            "Use at your own risk.": "Use at your own risk.",
            "Made with": "Made with",
            "for the Stremio community": "for the Stremio community",
            "What are we": "What are we",
            "watching": "watching",
            "take a peek": "Take a peek at some system stats and what we are watching in real time.",
            "Searches": "Searches",
            searches_last_24_hours_one: "1 search on the past 24 hours.",
            searches_last_24_hours_other: "{{count}} searches on the past 24 hours.",
            "Downloads": "Downloads",
            downloads_last_24_hours_one: "1 download on the past 24 hours.",
            downloads_last_24_hours_other: "{{count}} downloads on the past 24 hours.",
            "Last seen": "Last seen",
            last_seen_title: "{{title}}.",
        }
    },
    es: {
        translation: {
            "The definitive Spanish subtitles": "Los subtítulos definitivos en español",
            "Never Miss": "No Vuelvas A Perderte",
            "A Single Line": "Una Sola Línea",
            "Description": "Accede a la biblioteca definitiva de subtítulos en español de Subdivx directamente desde Stremio. Disfruta de tus películas y series favoritas con la sincronización perfecta.",
            "Install Now": "Instalar",
            "Buy Me a Coffee": "Invitame un café en cafecito.app",
            "Donate": "Donar",
            "Install manually": "O de forma manual copiando la URL del manifiesto:",
            "Disclaimer": "No estamos afiliados, asociados, autorizados, respaldados ni conectados oficialmente de ninguna manera con Subdivx.com.",
            "Use at your own risk.": "Usalo bajo su propio riesgo.",
            "Made with": "Hecho con",
            "for the Stremio community": "para la comunidad de Stremio",
            "What are we": "¿Que estamos",
            "watching": "viendo",
            "take a peek": "Echa un vistazo a algunas estadísticas del sistema y qué ven nuestros usuarios en tiempo real.",
            "Searches": "Búsquedas",
            searches_last_24_hours_one: "1 búsqueda en las últimas 24 horas.",
            searches_last_24_hours_other: "{{count}} búsquedas en las últimas 24 horas.",
            "Downloads": "Descargas",
            downloads_last_24_hours_one: "1 descarga en las últimas 24 horas.",
            downloads_last_24_hours_other: "{{count}} descargas en las últimas 24 horas.",
            "Last seen": "Recién visto",
            last_seen_title: "{{title}}.",
        }
    }
};


i18n
    .use(LanguageDetector)
    .use(initReactI18next) // passes i18n down to react-i18next
    .init({
        resources,
        supportedLngs: ['en', 'es'],

        interpolation: {
            escapeValue: false // react already safes from xss
        }
    });

export default i18n;