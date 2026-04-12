import { User, Bot } from 'lucide-react'
import styles from './ChatMessage.module.css'

export default function ChatMessage({ role, content, isStreaming }) {
  const isUser = role === 'user'

  return (
    <div className={`${styles.row} ${isUser ? styles.userRow : styles.assistantRow}`}>
      <div className={`${styles.avatar} ${isUser ? styles.userAvatar : styles.assistantAvatar}`}>
        {isUser ? <User size={16} /> : <Bot size={16} />}
      </div>
      <div className={`${styles.bubble} ${isUser ? styles.userBubble : styles.assistantBubble}`}>
        <div className={styles.content}>
          {content}
          {isStreaming && <span className={styles.cursor} />}
        </div>
      </div>
    </div>
  )
}
