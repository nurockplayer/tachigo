/**
 * @vitest-environment jsdom
 */
import React from 'react'
import { render, screen } from '@testing-library/react'
import { test } from 'vitest'

test('testing-library can render a React element in tachimint', () => {
  render(React.createElement('h1', null, 'Tachimint smoke test'))

  screen.getByRole('heading', { name: 'Tachimint smoke test' })
})
