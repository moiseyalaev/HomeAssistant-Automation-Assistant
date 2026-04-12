import { Zap, Lightbulb, Thermometer, Shield } from 'lucide-react'
import styles from './EmptyState.module.css'

const suggestions = [
  { icon: Lightbulb, text: 'Turn on the living room lights at sunset' },
  { icon: Thermometer, text: 'Set AC to 72 when I get home' },
  { icon: Shield, text: 'Lock doors and arm alarm at midnight' },
]

export default function EmptyState({ onSuggestion }) {
  return (
    <div className={styles.container}>
      <div className={styles.orbContainer}>
        <div className={styles.orb}>
          <Zap size={32} />
        </div>
        <div className={styles.orbRing} />
        <div className={styles.orbRing2} />
      </div>

      <h2 className={styles.title}>What would you like to automate?</h2>
      <p className={styles.subtitle}>
        Describe what you want in plain English and I'll build the automation for your Home Assistant.
      </p>

      <div className={styles.suggestions}>
        {suggestions.map(({ icon: Icon, text }) => (
          <button
            key={text}
            className={styles.suggestion}
            onClick={() => onSuggestion(text)}
          >
            <Icon size={16} className={styles.suggestionIcon} />
            <span>{text}</span>
          </button>
        ))}
      </div>
    </div>
  )
}
