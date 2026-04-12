import styles from './LoadingIndicator.module.css'

export default function LoadingIndicator() {
  return (
    <div className={styles.wrapper}>
      <div className={styles.orb}>
        <div className={styles.ring} />
        <div className={styles.core} />
      </div>
      <div className={styles.text}>
        <span className={styles.label}>Thinking</span>
        <span className={styles.dots}>
          <span className={styles.dot} />
          <span className={styles.dot} />
          <span className={styles.dot} />
        </span>
      </div>
    </div>
  )
}
