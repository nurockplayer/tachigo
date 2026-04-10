import { renderHook, waitFor } from '@testing-library/react'

import { useSound } from './useSound'

class MockAudioParam {
  setValueAtTime = vi.fn()
  linearRampToValueAtTime = vi.fn()
  exponentialRampToValueAtTime = vi.fn()
}

class MockAudioNode {
  connect = vi.fn()
}

class MockOscillatorNode extends MockAudioNode {
  type = 'sine'
  frequency = new MockAudioParam()
  start = vi.fn()
  stop = vi.fn()
}

class MockGainNode extends MockAudioNode {
  gain = new MockAudioParam()
}

class MockBufferSourceNode extends MockAudioNode {
  buffer: unknown = null
  start = vi.fn()
  stop = vi.fn()
}

class MockBiquadFilterNode extends MockAudioNode {
  type = 'bandpass'
  frequency = new MockAudioParam()
  Q = new MockAudioParam()
}

class MockAudioContext {
  static instances: MockAudioContext[] = []

  currentTime = 0
  sampleRate = 44100
  state: AudioContextState = 'running'
  destination = {}
  resume = vi.fn(async () => undefined)
  createOscillator = vi.fn(() => new MockOscillatorNode())
  createGain = vi.fn(() => new MockGainNode())
  createBuffer = vi.fn((_channels: number, length: number) => ({
    getChannelData: () => new Float32Array(length),
  }))
  createBufferSource = vi.fn(() => new MockBufferSourceNode())
  createBiquadFilter = vi.fn(() => new MockBiquadFilterNode())

  constructor() {
    MockAudioContext.instances.push(this)
  }
}

describe('useSound bridge behavior', () => {
  const query = vi.fn()
  const sendMessage = vi.fn()

  beforeEach(() => {
    MockAudioContext.instances = []
    query.mockResolvedValue([{ id: 123 }])
    sendMessage.mockResolvedValue(undefined)

    vi.stubGlobal('AudioContext', MockAudioContext)
    vi.stubGlobal('chrome', {
      tabs: {
        query,
        sendMessage,
      },
    })
  })

  it('uses the tab audio bridge without local fallback when bridge delivery succeeds', async () => {
    const { result } = renderHook(() => useSound())

    result.current.playMiningClick()

    await waitFor(() => {
      expect(sendMessage).toHaveBeenCalledTimes(1)
    })

    expect(result.current.bridgeStatus).toBe('ready')
    expect(MockAudioContext.instances).toHaveLength(0)
  })

  it('falls back to local playback and marks the bridge unsupported when bridge delivery fails', async () => {
    sendMessage.mockRejectedValue(new Error('Cannot access contents of the page'))

    const { result } = renderHook(() => useSound())

    result.current.playMiningClick()

    await waitFor(() => {
      expect(result.current.bridgeStatus).toBe('unsupported')
    })

    expect(sendMessage).toHaveBeenCalled()
    expect(MockAudioContext.instances).toHaveLength(1)
    expect(MockAudioContext.instances[0]?.createOscillator).toHaveBeenCalled()
  })
})
