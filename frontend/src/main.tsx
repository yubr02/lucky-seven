import React from 'react'
import ReactDOM from 'react-dom/client'
import StartScreen from './features/game/StartScreen'
import './styles/global.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <StartScreen />
  </React.StrictMode>,
)