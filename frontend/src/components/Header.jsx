import { Zap, RotateCcw, Volume2, VolumeX } from 'lucide-react'
import styles from './Header.module.css'

export default function Header({ onReset, voiceEnabled, onToggleVoice, hasRecognition }) {
  return (
    <header className={styles.header}>
      <div className={styles.left}>
        <div className={styles.logo}>
          <Zap size={20} />
        </div>
        <div className={styles.titleGroup}>
          <h1 className={styles.title}>Automation Assistant</h1>
          <span className={styles.badge}>Home Assistant</span>
        </div>
      </div>
      <div className={styles.actions}>
        {hasRecognition && (
          <button
            className={`${styles.btn} ${voiceEnabled ? styles.active : ''}`}
            onClick={onToggleVoice}
            title={voiceEnabled ? 'Disable voice output' : 'Enable voice output'}
          >
            {voiceEnabled ? <Volume2 size={16} /> : <VolumeX size={16} />}
          </button>
        )}
        <button className={styles.btn} onClick={onReset} title="New conversation">
          <RotateCcw size={16} />
        </button>
      </div>
    </header>
  )
}
