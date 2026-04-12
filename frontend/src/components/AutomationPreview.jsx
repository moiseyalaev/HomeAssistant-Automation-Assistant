import { Check, X, Cog, FileCode } from 'lucide-react'
import styles from './AutomationPreview.module.css'

export default function AutomationPreview({ automation, onConfirm, onCancel }) {
  if (!automation) return null

  return (
    <div className={styles.overlay}>
      <div className={styles.panel}>
        <div className={styles.header}>
          <div className={styles.headerLeft}>
            <div className={styles.icon}>
              <Cog size={18} />
            </div>
            <div>
              <h3 className={styles.title}>Proposed Automation</h3>
              <p className={styles.subtitle}>{automation.name || 'Untitled'}</p>
            </div>
          </div>
          <button className={styles.close} onClick={onCancel}>
            <X size={16} />
          </button>
        </div>

        <div className={styles.yamlSection}>
          <div className={styles.yamlHeader}>
            <FileCode size={14} />
            <span>automation.yaml</span>
          </div>
          <pre className={styles.yaml}><code>{automation.yaml}</code></pre>
        </div>

        {automation.description && (
          <p className={styles.description}>{automation.description}</p>
        )}

        <div className={styles.actions}>
          <button className={styles.cancelBtn} onClick={onCancel}>
            <X size={15} />
            Cancel
          </button>
          <button className={styles.confirmBtn} onClick={onConfirm}>
            <Check size={15} />
            Deploy to Home Assistant
          </button>
        </div>

        <div className={styles.glow} />
      </div>
    </div>
  )
}
