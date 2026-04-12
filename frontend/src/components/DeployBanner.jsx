import styles from './DeployBanner.module.css'

export default function DeployBanner() {
  return (
    <div className={styles.banner}>
      <div className={styles.pulse} />
      <span className={styles.text}>
        Deploying automation to Home Assistant — writing config and reloading…
      </span>
    </div>
  )
}
