import { CalculatorForm } from './components/CalculatorForm'
import styles from './App.module.css'

export default function App() {
  return (
    <main className={styles.page}>
      <header className={styles.header}>
        <h1 className={styles.title}>Calculator</h1>
        <p className={styles.subtitle}>
          Every result is computed by the Go service using exact decimal
          arithmetic, so 0.1 + 0.2 is 0.3.
        </p>
      </header>

      <CalculatorForm />

      <footer className={styles.footer}>
        Results are exact to 16 significant digits. Division and square roots are
        rounded; addition, subtraction and multiplication are not.
      </footer>
    </main>
  )
}
