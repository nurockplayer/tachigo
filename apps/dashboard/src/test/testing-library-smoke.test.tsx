import { render, screen } from '@testing-library/react'
import { test } from 'vitest'

test('testing-library can render a dashboard element', () => {
  render(<h1>Tachigo Dashboard smoke test</h1>)

  screen.getByRole('heading', { name: 'Tachigo Dashboard smoke test' })
})
