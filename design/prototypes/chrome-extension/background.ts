chrome.runtime.onInstalled.addListener(() => {
  chrome.sidePanel?.setPanelBehavior?.({ openPanelOnActionClick: true }).catch(() => {});
});

chrome.action.onClicked.addListener((tab) => {
  if (tab.windowId == null || tab.windowId === chrome.windows.WINDOW_ID_NONE) return;
  chrome.sidePanel?.open?.({ windowId: tab.windowId }).catch(() => {});
});
