'use client'

import { createContext, useContext, useEffect, useState } from 'react'

type Language = 'en' | 'es' | 'fr' | 'de'

interface LanguageContextType {
  language: Language
  setLanguage: (language: Language) => void
  t: (key: string) => string
}

const translations = {
  en: {
    'settings.title': 'Settings',
    'settings.profile': 'Profile',
    'settings.security': 'Security',
    'settings.notifications': 'Notifications',
    'settings.privacy': 'Privacy',
    'settings.appearance': 'Appearance',
    'settings.theme': 'Theme',
    'settings.language': 'Language',
    'settings.timezone': 'Timezone',
    'settings.save': 'Save',
    'settings.light': 'Light',
    'settings.dark': 'Dark',
    'settings.system': 'System',
    'settings.english': 'English',
    'settings.spanish': 'Spanish',
    'settings.french': 'French',
    'settings.german': 'German',
    'settings.utc': 'UTC',
    'settings.est': 'Eastern Time',
    'settings.pst': 'Pacific Time',
    'settings.gmt': 'Greenwich Mean Time',
    'settings.saved': 'Settings saved!',
  },
  es: {
    'settings.title': 'Configuración',
    'settings.profile': 'Perfil',
    'settings.security': 'Seguridad',
    'settings.notifications': 'Notificaciones',
    'settings.privacy': 'Privacidad',
    'settings.appearance': 'Apariencia',
    'settings.theme': 'Tema',
    'settings.language': 'Idioma',
    'settings.timezone': 'Zona horaria',
    'settings.save': 'Guardar',
    'settings.light': 'Claro',
    'settings.dark': 'Oscuro',
    'settings.system': 'Sistema',
    'settings.english': 'Inglés',
    'settings.spanish': 'Español',
    'settings.french': 'Francés',
    'settings.german': 'Alemán',
    'settings.utc': 'UTC',
    'settings.est': 'Hora del Este',
    'settings.pst': 'Hora del Pacífico',
    'settings.gmt': 'Hora de Greenwich',
    'settings.saved': '¡Configuración guardada!',
  },
  fr: {
    'settings.title': 'Paramètres',
    'settings.profile': 'Profil',
    'settings.security': 'Sécurité',
    'settings.notifications': 'Notifications',
    'settings.privacy': 'Confidentialité',
    'settings.appearance': 'Apparence',
    'settings.theme': 'Thème',
    'settings.language': 'Langue',
    'settings.timezone': 'Fuseau horaire',
    'settings.save': 'Enregistrer',
    'settings.light': 'Clair',
    'settings.dark': 'Sombre',
    'settings.system': 'Système',
    'settings.english': 'Anglais',
    'settings.spanish': 'Espagnol',
    'settings.french': 'Français',
    'settings.german': 'Allemand',
    'settings.utc': 'UTC',
    'settings.est': 'Heure de l\'Est',
    'settings.pst': 'Heure du Pacifique',
    'settings.gmt': 'Heure de Greenwich',
    'settings.saved': 'Paramètres enregistrés !',
  },
  de: {
    'settings.title': 'Einstellungen',
    'settings.profile': 'Profil',
    'settings.security': 'Sicherheit',
    'settings.notifications': 'Benachrichtigungen',
    'settings.privacy': 'Datenschutz',
    'settings.appearance': 'Erscheinungsbild',
    'settings.theme': 'Design',
    'settings.language': 'Sprache',
    'settings.timezone': 'Zeitzone',
    'settings.save': 'Speichern',
    'settings.light': 'Hell',
    'settings.dark': 'Dunkel',
    'settings.system': 'System',
    'settings.english': 'Englisch',
    'settings.spanish': 'Spanisch',
    'settings.french': 'Französisch',
    'settings.german': 'Deutsch',
    'settings.utc': 'UTC',
    'settings.est': 'Ostküstenzeit',
    'settings.pst': 'Pazifikküstenzeit',
    'settings.gmt': 'Greenwich-Zeit',
    'settings.saved': 'Einstellungen gespeichert!',
  },
}

const LanguageContext = createContext<LanguageContextType | undefined>(undefined)

export function LanguageProvider({ children }: { children: React.ReactNode }) {
  const [language, setLanguage] = useState<Language>('en')

  useEffect(() => {
    // Load language from localStorage on mount
    const savedLanguage = localStorage.getItem('language') as Language
    if (savedLanguage && ['en', 'es', 'fr', 'de'].includes(savedLanguage)) {
      setLanguage(savedLanguage)
    }
  }, [])

  useEffect(() => {
    // Save to localStorage when language changes
    localStorage.setItem('language', language)
  }, [language])

  const t = (key: string): string => {
    return translations[language][key as keyof typeof translations[typeof language]] || key
  }

  return (
    <LanguageContext.Provider value={{ language, setLanguage, t }}>
      {children}
    </LanguageContext.Provider>
  )
}

export function useLanguage() {
  const context = useContext(LanguageContext)
  if (context === undefined) {
    throw new Error('useLanguage must be used within a LanguageProvider')
  }
  return context
}
