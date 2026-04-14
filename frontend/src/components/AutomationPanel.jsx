import { useState, useEffect, useRef } from 'react'
import { Cog, X, Copy, Check, Rocket, ShieldCheck, AlertTriangle, AlertCircle, ChevronDown, ChevronRight, Wand2 } from 'lucide-react'
import styles from './AutomationPanel.module.css'

export default function AutomationPanel({ automation, onConfirm, onDismiss, onYamlChange, onFixWithAI, isConfirming }) {
  const [yaml, setYaml] = useState(automation.yaml)
  const [copied, setCopied] = useState(false)
  const [validation, setValidation] = useState(null)   // null | { valid, errors, warnings, missing_entities }
  const [isValidating, setIsValidating] = useState(false)
  const [showWarnings, setShowWarnings] = useState(true)
  const textareaRef = useRef(null)

  // When the LLM pushes a new automation, reset to it
  useEffect(() => {
    setYaml(automation.yaml)
    setValidation(null)
  }, [automation.id, automation.yaml])

  // Auto-grow textarea height
  useEffect(() => {
    const el = textareaRef.current
    if (!el) return
    el.style.height = 'auto'
    el.style.height = el.scrollHeight + 'px'
  }, [yaml])

  const handleChange = (e) => {
    const val = e.target.value
    setYaml(val)
    setValidation(null)
    onYamlChange(val)
  }

  const copyYaml = async () => {
    try {
      await navigator.clipboard.writeText(yaml)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {}
  }

  const runValidation = async () => {
    setIsValidating(true)
    setValidation(null)
    try {
      const resp = await fetch('/api/validate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ yaml }),
      })
      const result = await resp.json()
      setValidation(result)
      setShowWarnings(true)
    } catch (err) {
      setValidation({ valid: false, errors: [`Validation request failed: ${err.message}`], warnings: [], missing_entities: [] })
    } finally {
      setIsValidating(false)
    }
  }

  const handleTabKey = (e) => {
    if (e.key !== 'Tab') return
    e.preventDefault()
    const el = e.target
    const start = el.selectionStart
    const end = el.selectionEnd
    const newVal = yaml.slice(0, start) + '  ' + yaml.slice(end)
    setYaml(newVal)
    onYamlChange(newVal)
    // Restore cursor position after React re-render
    requestAnimationFrame(() => {
      el.selectionStart = el.selectionEnd = start + 2
    })
  }

  const isDirty = yaml !== automation.yaml

  const validationIcon = () => {
    if (!validation) return null
    if (validation.valid && validation.warnings.length === 0) return <Check size={13} className={styles.iconOk} />
    if (validation.valid) return <AlertTriangle size={13} className={styles.iconWarn} />
    return <AlertCircle size={13} className={styles.iconErr} />
  }

  return (
    <div className={styles.panel}>
      <div className={styles.header}>
        <div className={styles.headerLeft}>
          <div className={styles.icon}>
            <Cog size={16} />
          </div>
          <div>
            <h3 className={styles.title}>{automation.name || 'Proposed Automation'}</h3>
            <span className={styles.badge}>
              {isDirty ? 'Edited locally' : 'Ready to deploy'}
            </span>
          </div>
        </div>
        <button className={styles.closeBtn} onClick={onDismiss} title="Dismiss panel">
          <X size={15} />
        </button>
      </div>

      {automation.description && (
        <p className={styles.description}>{automation.description}</p>
      )}

      <div className={styles.yamlBlock}>
        <div className={styles.yamlHeader}>
          <span className={styles.filename}>automation.yaml</span>
          <div className={styles.yamlActions}>
            {isDirty && <span className={styles.dirtyPill}>unsaved edits</span>}
            <button className={styles.iconBtn} onClick={copyYaml} title="Copy YAML">
              {copied ? <Check size={13} /> : <Copy size={13} />}
            </button>
          </div>
        </div>

        <textarea
          ref={textareaRef}
          className={styles.editor}
          value={yaml}
          onChange={handleChange}
          onKeyDown={handleTabKey}
          spellCheck={false}
          autoComplete="off"
          autoCorrect="off"
          autoCapitalize="off"
        />
      </div>

      {/* Validation result */}
      {validation && (
        <div className={`${styles.validResult} ${validation.valid ? styles.validOk : styles.validFail}`}>
          <div className={styles.validTitleRow}>
            <div className={styles.validTitle}>
              {validationIcon()}
              {validation.valid
                ? validation.warnings.length === 0 ? 'Looks good' : 'Valid with warnings'
                : `${validation.errors.length} error${validation.errors.length !== 1 ? 's' : ''} found`
              }
            </div>
            {(validation.errors.length > 0 || validation.warnings.length > 0) && (
              <button
                className={styles.fixBtn}
                onClick={() => {
                  const issues = [
                    ...validation.errors.map(e => `Error: ${e}`),
                    ...validation.warnings.map(w => `Warning: ${w}`),
                  ].join('\n')
                  onFixWithAI(`The YAML validator found the following issues — please fix them:\n\n${issues}`)
                }}
              >
                <Wand2 size={11} /> Ask Claude to fix
              </button>
            )}
          </div>
          {validation.errors.map((e, i) => (
            <p key={i} className={styles.validError}><AlertCircle size={11} /> {e}</p>
          ))}
          {validation.warnings.length > 0 && (
            <>
              <button className={styles.warnToggle} onClick={() => setShowWarnings(v => !v)}>
                {showWarnings ? <ChevronDown size={11} /> : <ChevronRight size={11} />}
                {validation.warnings.length} warning{validation.warnings.length !== 1 ? 's' : ''}
              </button>
              {showWarnings && validation.warnings.map((w, i) => (
                <p key={i} className={styles.validWarn}><AlertTriangle size={11} /> {w}</p>
              ))}
            </>
          )}
        </div>
      )}

      <div className={styles.footer}>
        <button
          className={`${styles.validateBtn} ${isValidating ? styles.validating : ''}`}
          onClick={runValidation}
          disabled={isValidating}
        >
          {isValidating
            ? <><span className={styles.spinner} /> Checking…</>
            : <><ShieldCheck size={14} /> Validate YAML</>
          }
        </button>
        <button
          className={styles.deployBtn}
          onClick={onConfirm}
          disabled={isConfirming}
        >
          {isConfirming
            ? <><span className={styles.spinner} /> Deploying…</>
            : <><Rocket size={14} /> Deploy to Home Assistant</>
          }
        </button>
      </div>

      <div className={styles.glowTop} />
    </div>
  )
}
