import { useState } from 'react'

function App() {
  // 這是假資料，之後會對接 5lime 負責的後端 API
  const agencyList = [
    { id: 1, name: 'Tachigo 經紀公司', token: 'TGO', streamers: 5 },
    { id: 2, name: 'Newbie Agency', token: 'NEW', streamers: 2 },
  ];

  return (
    <div style={{ padding: '20px', fontFamily: 'sans-serif', color: '#333' }}>
      <header style={{ borderBottom: '2px solid #646cff', paddingBottom: '10px', marginBottom: '20px' }}>
        <h1>Tachigo Dashboard — 管理後台</h1>
        <p style={{ color: '#666' }}>功能 6：Agency / Streamer 權限與資料管理</p>
      </header>

      <section>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <h2>Agency 列表 (功能 2)</h2>
          <button style={{ 
            padding: '8px 16px', 
            backgroundColor: '#646cff', 
            color: 'white', 
            border: 'none', 
            borderRadius: '4px',
            cursor: 'pointer'
          }}>
            + 新增 Agency
          </button>
        </div>

        <table style={{ width: '100%', borderCollapse: 'collapse', marginTop: '20px', boxShadow: '0 2px 5px rgba(0,0,0,0.1)' }}>
          <thead>
            <tr style={{ backgroundColor: '#f4f4f4', textAlign: 'left' }}>
              <th style={{ padding: '12px', border: '1px solid #ddd' }}>ID</th>
              <th style={{ padding: '12px', border: '1px solid #ddd' }}>公司名稱</th>
              <th style={{ padding: '12px', border: '1px solid #ddd' }}>代幣符號</th>
              <th style={{ padding: '12px', border: '1px solid #ddd' }}>實況主數量</th>
              <th style={{ padding: '12px', border: '1px solid #ddd' }}>操作</th>
            </tr>
          </thead>
          <tbody>
            {agencyList.map((agency) => (
              <tr key={agency.id} style={{ borderBottom: '1px solid #eee' }}>
                <td style={{ padding: '12px', border: '1px solid #ddd' }}>{agency.id}</td>
                <td style={{ padding: '12px', border: '1px solid #ddd' }}>{agency.name}</td>
                <td style={{ padding: '12px', border: '1px solid #ddd' }}>{agency.token}</td>
                <td style={{ padding: '12px', border: '1px solid #ddd' }}>{agency.streamers}</td>
                <td style={{ padding: '12px', border: '1px solid #ddd' }}>
                  <button style={{ marginRight: '5px', cursor: 'pointer' }}>編輯</button>
                  <button style={{ color: 'red', cursor: 'pointer' }}>刪除</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <footer style={{ marginTop: '40px', fontSize: '12px', color: '#999' }}>
        <p>當前分支：feat/dashboard-setup | 參考 Issue #18</p>
      </footer>
    </div>
  );
}

export default App;