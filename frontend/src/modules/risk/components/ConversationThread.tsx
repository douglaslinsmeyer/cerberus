import { useState } from 'react'
import { PaperAirplaneIcon, UserCircleIcon } from '@heroicons/react/24/outline'
import { ConversationMessage } from '@/services/api'
import { format } from 'date-fns'

interface ConversationThreadProps {
  messages: ConversationMessage[]
  onSendMessage: (message: string) => void
  isLoading?: boolean
}

export function ConversationThread({ messages, onSendMessage, isLoading = false }: ConversationThreadProps) {
  const [newMessage, setNewMessage] = useState('')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (newMessage.trim()) {
      onSendMessage(newMessage.trim())
      setNewMessage('')
    }
  }

  const formatDate = (dateString: string) => {
    try {
      return format(new Date(dateString), 'MMM d, yyyy h:mm a')
    } catch {
      return dateString
    }
  }

  // Simple markdown-like rendering (basic support)
  const renderMessage = (text: string) => {
    // Convert **bold** to <strong>
    let html = text.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    // Convert *italic* to <em>
    html = html.replace(/\*(.+?)\*/g, '<em>$1</em>')
    // Convert line breaks
    html = html.replace(/\n/g, '<br />')

    return <div dangerouslySetInnerHTML={{ __html: html }} />
  }

  return (
    <div className="bg-white rounded-lg shadow">
      {/* Messages List */}
      <div className="max-h-96 overflow-y-auto p-6 space-y-4">
        {messages.length === 0 ? (
          <div className="text-center text-gray-500 py-8">
            <p>No messages yet. Start the conversation!</p>
          </div>
        ) : (
          messages.map((message) => (
            <div key={message.message_id} className="flex items-start space-x-3">
              <UserCircleIcon className="h-8 w-8 text-gray-400 flex-shrink-0" />

              <div className="flex-1">
                <div className="flex items-center space-x-2">
                  <span className="text-sm font-medium text-gray-900">User</span>
                  <span className="text-xs text-gray-500">{formatDate(message.created_at)}</span>
                  {message.edited_at?.Valid && (
                    <span className="text-xs text-gray-400">(edited)</span>
                  )}
                </div>

                <div className="mt-1 text-sm text-gray-700 prose prose-sm max-w-none">
                  {message.message_format === 'markdown' ? (
                    renderMessage(message.message_text)
                  ) : (
                    <p>{message.message_text}</p>
                  )}
                </div>
              </div>
            </div>
          ))
        )}

        {isLoading && (
          <div className="flex items-center justify-center py-4">
            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600"></div>
          </div>
        )}
      </div>

      {/* Message Input */}
      <div className="border-t border-gray-200 p-4">
        <form onSubmit={handleSubmit} className="flex items-end space-x-3">
          <div className="flex-1">
            <textarea
              value={newMessage}
              onChange={(e) => setNewMessage(e.target.value)}
              placeholder="Type your message... (supports *italic* and **bold**)"
              rows={3}
              className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
              disabled={isLoading}
            />
          </div>

          <button
            type="submit"
            disabled={!newMessage.trim() || isLoading}
            className="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <PaperAirplaneIcon className="h-5 w-5" />
          </button>
        </form>

        <p className="mt-2 text-xs text-gray-500">
          Tip: Use *italic* for emphasis and **bold** for strong emphasis
        </p>
      </div>
    </div>
  )
}
