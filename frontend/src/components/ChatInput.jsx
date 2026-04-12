import { useState, useRef, useEffect } from 'react'
import { Send, Mic, MicOff, Square } from 'lucide-react'
import styles from './ChatInput.module.css'

export default function ChatInput({
  onSend,
  disabled,
  isDeploying,
  isListening,
  onStartListening,
  onStopListening,
  hasRecognition,
}) {
  const [text, setText] = useState('')
  const inputRef = useRef(null)

  useEffect(() => {
    if (!disabled) inputRef.current?.focus()
  }, [disabled])

  const handleSubmit = (e) => {
    e.preventDefault()
    const trimmed = text.trim()
    if (!trimmed || disabled) return
    onSend(trimmed)
    setText('')
  }

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      handleSubmit(e)
    }
  }

  return (
    <form className={styles.form} onSubmit={handleSubmit}>
      <div className={styles.inputWrapper}>
        <textarea
          ref={inputRef}
          className={styles.input}
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={
            isListening ? 'Listening...'
            : isDeploying ? 'Deploying to Home Assistant — please wait…'
            : disabled ? 'Waiting for response…'
            : 'Describe an automation...'
          }
          disabled={disabled}
          rows={1}
        />
        <div className={styles.buttons}>
          {hasRecognition && (
            <button
              type="button"
              className={`${styles.btn} ${isListening ? styles.listening : ''}`}
              onClick={isListening ? onStopListening : onStartListening}
              title={isListening ? 'Stop listening' : 'Voice input'}
            >
              {isListening ? <Square size={14} /> : <Mic size={14} />}
            </button>
          )}
          <button
            type="submit"
            className={`${styles.btn} ${styles.sendBtn}`}
            disabled={!text.trim() || disabled}
          >
            <Send size={14} />
          </button>
        </div>
      </div>
      <p className={styles.hint}>
        {isDeploying ? 'Writing automation to Home Assistant and reloading…' : 'Press Enter to send · Shift+Enter for new line'}
      </p>
    </form>
  )
}
