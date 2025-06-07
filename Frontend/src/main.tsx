import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import ChessInterface from './Chess.tsx'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ChessInterface />
  </StrictMode>,
)
