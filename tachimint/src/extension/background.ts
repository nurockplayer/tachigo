async function enableSidePanelAction() {
  if (!chrome.sidePanel?.setPanelBehavior) {
    return
  }

  try {
    await chrome.sidePanel.setPanelBehavior({ openPanelOnActionClick: true })
  } catch (error) {
    console.warn('Failed to enable side panel action behavior', error)
  }
}

chrome.runtime.onInstalled.addListener(() => {
  void enableSidePanelAction()
})

chrome.runtime.onStartup.addListener(() => {
  void enableSidePanelAction()
})

// Message listener stub — wired in PR 3
chrome.runtime.onMessage.addListener(() => undefined)
