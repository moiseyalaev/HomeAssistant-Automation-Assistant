import { useState } from 'react'
import styles from './StatusBanner.module.css'

export default function StatusBanner() {
  const [dismissed, setDismissed] = useState(false)

  if (dismissed) return null

  return (
    <div className={styles.banner}>
      <span className={styles.icon}>⚠</span>
      <span className={styles.text}>
        Device context unavailable — automations will be created without area or device grouping
      </span>
      <button
        className={styles.close}
        onClick={() => setDismissed(true)}
        title="Dismiss"
        aria-label="Dismiss warning"
      >
        ✕
      </button>
    </div>
  )
}
