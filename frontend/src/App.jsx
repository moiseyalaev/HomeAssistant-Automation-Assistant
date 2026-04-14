import { useRef, useEffect, useCallback, useState } from 'react'
import useChat from './hooks/useChat'
import useVoice from './hooks/useVoice'
import Header from './components/Header'
import EmptyState from './components/EmptyState'
import ChatMessage from './components/ChatMessage'
import LoadingIndicator from './components/LoadingIndicator'
import DeployBanner from './components/DeployBanner'
import AutomationPanel from './components/AutomationPanel'
import ChatInput from './components/ChatInput'
import styles from './App.module.css'

// Split direction: 'horizontal' = chat|panel, 'vertical' = chat/panel
const DEFAULT_RATIO = 55   // percent for the chat pane
const MIN_RATIO = 25
const MAX_RATIO = 80

export default function App() {
  const chat = useChat()
  const scrollRef = useRef(null)
  const prevMessageCount = useRef(0)

  // Split-pane state
  const [splitDir, setSplitDir] = useState('horizontal')
  const [splitRatio, setSplitRatio] = useState(DEFAULT_RATIO)
  const splitContainerRef = useRef(null)
  const dragging = useRef(false)

  const voice = useVoice({ onTranscript: chat.send })

  // Scroll to bottom when a new message is added (non-streaming) or continuously
  // during streaming via the interval. A single effect avoids redundant scrollTo
  // calls that occurred when both fired simultaneously during streaming.
  useEffect(() => {
    if (chat.isStreaming) {
      const id = setInterval(() => {
        scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: 'smooth' })
      }, 80)
      return () => clearInterval(id)
    }
    // Not streaming — scroll once when message count grows (e.g. user message added)
    if (chat.messages.length > prevMessageCount.current) {
      scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: 'smooth' })
    }
    prevMessageCount.current = chat.messages.length
  }, [chat.isStreaming, chat.messages])

  // Speak the last assistant message once streaming completes
  useEffect(() => {
    if (!voice.voiceEnabled || chat.isStreaming) return
    const last = chat.messages[chat.messages.length - 1]
    if (last?.role === 'assistant' && last.content) {
      voice.speak(last.content)
    }
  }, [chat.isStreaming, voice.voiceEnabled])

  // Draggable divider — pointer events on document for smooth tracking
  const startDrag = useCallback((e) => {
    e.preventDefault()
    dragging.current = true

    const onMove = (e) => {
      if (!dragging.current || !splitContainerRef.current) return
      const rect = splitContainerRef.current.getBoundingClientRect()
      let ratio
      if (splitDir === 'horizontal') {
        ratio = ((e.clientX - rect.left) / rect.width) * 100
      } else {
        ratio = ((e.clientY - rect.top) / rect.height) * 100
      }
      setSplitRatio(Math.min(Math.max(ratio, MIN_RATIO), MAX_RATIO))
    }

    const onUp = () => {
      dragging.current = false
      document.removeEventListener('mousemove', onMove)
      document.removeEventListener('mouseup', onUp)
    }

    document.addEventListener('mousemove', onMove)
    document.addEventListener('mouseup', onUp)
  }, [splitDir])

  const toggleDirection = useCallback(() => {
    setSplitDir(d => d === 'horizontal' ? 'vertical' : 'horizontal')
    setSplitRatio(DEFAULT_RATIO)
  }, [])

  const showPanel = !!chat.pendingAutomation
  const isEmpty = chat.messages.length === 0

  const chatPane = (
    <div className={styles.chatPane}>
      <div className={styles.chatScroll} ref={scrollRef}>
        {isEmpty ? (
          <EmptyState onSuggestion={chat.send} />
        ) : (
          <div className={styles.messages}>
            {chat.messages.map((msg, i) => (
              <ChatMessage key={i} {...msg} />
            ))}
            {chat.isConfirming && <DeployBanner />}
            {chat.isLoading && !chat.isConfirming && <LoadingIndicator />}
            {chat.error && <div className={styles.error}>{chat.error}</div>}
          </div>
        )}
      </div>
      <ChatInput
        onSend={chat.send}
        disabled={chat.isLoading || chat.isStreaming}
        isDeploying={chat.isConfirming}
        isListening={voice.isListening}
        onStartListening={voice.startListening}
        onStopListening={voice.stopListening}
        hasRecognition={voice.hasRecognition}
      />
    </div>
  )

  return (
    <>
      <Header
        onReset={chat.reset}
        voiceEnabled={voice.voiceEnabled}
        onToggleVoice={() => voice.setVoiceEnabled(v => !v)}
        hasRecognition={voice.hasRecognition}
      />

      {showPanel ? (
        <div
          ref={splitContainerRef}
          className={styles.splitContainer}
          style={{ flexDirection: splitDir === 'horizontal' ? 'row' : 'column' }}
        >
          {/* Chat pane */}
          <div
            className={styles.splitChat}
            style={
              splitDir === 'horizontal'
                ? { width: `${splitRatio}%` }
                : { height: `${splitRatio}%` }
            }
          >
            {chatPane}
          </div>

          {/* Draggable divider */}
          <div
            className={`${styles.divider} ${splitDir === 'horizontal' ? styles.dividerH : styles.dividerV}`}
            onMouseDown={startDrag}
          >
            <button
              className={styles.dirToggle}
              onMouseDown={e => e.stopPropagation()}
              onClick={toggleDirection}
              title={splitDir === 'horizontal' ? 'Switch to top / bottom' : 'Switch to left / right'}
            >
              {splitDir === 'horizontal' ? '⇅' : '⇄'}
            </button>
          </div>

          {/* Automation panel */}
          <div className={styles.splitPanel}>
            <AutomationPanel
              automation={chat.pendingAutomation}
              onConfirm={chat.confirm}
              onDismiss={chat.dismissAutomation}
              onYamlChange={chat.updateEditedYaml}
              onFixWithAI={chat.send}
              isConfirming={chat.isConfirming}
            />
          </div>
        </div>
      ) : (
        <div className={styles.fullChat}>
          {chatPane}
        </div>
      )}
    </>
  )
}
