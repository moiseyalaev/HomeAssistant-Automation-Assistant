import { useState, useCallback, useRef } from 'react'

export default function useChat() {
  const [messages, setMessages] = useState([])
  const [isLoading, setIsLoading] = useState(false)
  const [isStreaming, setIsStreaming] = useState(false)
  const [pendingAutomation, setPendingAutomation] = useState(null)
  const [isConfirming, setIsConfirming] = useState(false)
  const [error, setError] = useState(null)
  const sessionIdRef = useRef(null)
  const abortRef = useRef(null)
  // Tracks user edits to the YAML panel — separate from the LLM-proposed version
  const editedYamlRef = useRef(null)

  const send = useCallback(async (message, confirmed = false) => {
    setError(null)
    if (confirmed) setIsConfirming(true)

    if (message) {
      setMessages(prev => [...prev, { role: 'user', content: message }])
    }

    setIsLoading(true)
    setIsStreaming(false)

    const body = { message: message || '', confirmed }
    if (sessionIdRef.current) body.session_id = sessionIdRef.current
    // Always send the current edited YAML so the backend (and therefore Claude)
    // works from the user's latest version, not the original LLM proposal
    if (editedYamlRef.current !== null) body.edited_yaml = editedYamlRef.current

    abortRef.current = new AbortController()

    try {
      const resp = await fetch('/api/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
        signal: abortRef.current.signal,
      })

      if (!resp.ok) {
        const text = await resp.text()
        throw new Error(`Server error ${resp.status}: ${text}`)
      }

      const reader = resp.body.getReader()
      const decoder = new TextDecoder()
      let assistantText = ''
      let buffer = ''

      setMessages(prev => [...prev, { role: 'assistant', content: '', isStreaming: true }])
      setIsLoading(false)
      setIsStreaming(true)

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop()

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue
          try {
            const data = JSON.parse(line.slice(6))

            if (data.session_id) sessionIdRef.current = data.session_id

            if (data.text) {
              assistantText += data.text
              setMessages(prev => {
                const updated = [...prev]
                updated[updated.length - 1] = {
                  role: 'assistant',
                  content: assistantText,
                  isStreaming: true,
                }
                return updated
              })
            }

            if (data.automation) {
              // New proposal from LLM — reset edited YAML to the fresh proposal
              editedYamlRef.current = null
              setPendingAutomation(data.automation)
            }

            if (data.done) {
              setMessages(prev => {
                const updated = [...prev]
                updated[updated.length - 1] = { ...updated[updated.length - 1], isStreaming: false }
                return updated
              })
            }
          } catch {
            // skip malformed
          }
        }
      }

      setMessages(prev => {
        const updated = [...prev]
        if (updated.length > 0) {
          updated[updated.length - 1] = { ...updated[updated.length - 1], isStreaming: false }
        }
        return updated
      })
    } catch (err) {
      if (err.name !== 'AbortError') {
        setError(err.message)
        setMessages(prev => {
          if (prev.length > 0 && prev[prev.length - 1].role === 'assistant' && !prev[prev.length - 1].content) {
            return prev.slice(0, -1)
          }
          return prev
        })
      }
    } finally {
      setIsLoading(false)
      setIsStreaming(false)
      setIsConfirming(false)
      abortRef.current = null
    }
  }, [])

  // Called by AutomationPanel when user edits the YAML textarea
  const updateEditedYaml = useCallback((yaml) => {
    editedYamlRef.current = yaml
  }, [])

  const confirm = useCallback(() => {
    send('', true)
  }, [send])

  const dismissAutomation = useCallback(() => {
    setPendingAutomation(null)
    editedYamlRef.current = null
  }, [])

  const reset = useCallback(() => {
    setMessages([])
    setPendingAutomation(null)
    setIsConfirming(false)
    setError(null)
    sessionIdRef.current = null
    editedYamlRef.current = null
  }, [])

  return {
    messages,
    isLoading,
    isStreaming,
    isConfirming,
    pendingAutomation,
    error,
    send,
    confirm,
    dismissAutomation,
    updateEditedYaml,
    reset,
  }
}
