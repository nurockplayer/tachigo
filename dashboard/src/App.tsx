import { useState } from 'react'
import './index.css'

function App() {
  const [agencyList, setAgencyList] = useState([
    { id: 1, name: 'Tachigo 經紀公司', token: 'TGO', streamers: 5 },
    { id: 2, name: 'Newbie Agency', token: 'NEW', streamers: 2 },
  ]);

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [newAgency, setNewAgency] = useState({ name: '', token: '' });

  const handleAddAgency = () => {
    if (!newAgency.name || !newAgency.token) return alert("請填寫完整資訊");
    const id = agencyList.length + 1;
    setAgencyList([...agencyList, { ...newAgency, id, streamers: 0 }]);
    setNewAgency({ name: '', token: '' });
    setIsModalOpen(false);
  };

  return (
    <div className="min-h-screen bg-gray-50 p-8 font-sans text-gray-800">
      <header className="mb-8 border-b-2 border-indigo-600 pb-4">
        <h1 className="text-3xl font-bold text-indigo-700">Tachigo Dashboard</h1>
        <p className="mt-2 text-gray-500">功能 6：Agency / Streamer 管理後台</p>
      </header>

      <section className="rounded-lg bg-white p-6 shadow-md">
        <div className="mb-6 flex items-center justify-between">
          <h2 className="text-xl font-semibold text-gray-700">Agency 列表</h2>
          <button 
            onClick={() => setIsModalOpen(true)} 
            className="rounded bg-indigo-600 px-4 py-2 text-white shadow-sm transition-colors hover:bg-indigo-700 active:transform active:scale-95"
          >
            + 新增 Agency
          </button>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full border-collapse text-left">
            <thead>
              <tr className="border-b border-gray-200 bg-gray-50 text-sm font-semibold uppercase text-gray-600">
                <th className="p-4">ID</th>
                <th className="p-4">公司名稱</th>
                <th className="p-4">代幣符號</th>
                <th className="p-4">實況主</th>
                <th className="p-4">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {agencyList.map((agency) => (
                <tr key={agency.id} className="hover:bg-gray-50 transition-colors">
                  <td className="p-4 text-gray-500">{agency.id}</td>
                  <td className="p-4 font-medium">{agency.name}</td>
                  <td className="p-4">
                    <span className="rounded bg-indigo-100 px-2 py-1 text-xs font-bold text-indigo-700">
                      {agency.token}
                    </span>
                  </td>
                  <td className="p-4 text-gray-600">{agency.streamers}</td>
                  <td className="p-4">
                    <button className="mr-3 font-medium text-indigo-600 hover:text-indigo-900">編輯</button>
                    <button className="font-medium text-red-600 hover:text-red-900 border-none bg-transparent cursor-pointer">刪除</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      {/* 彈窗實作 (Modal) */}
      {isModalOpen && (
        <div className="fixed inset-0 z-[9999] flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="w-full max-w-md rounded-xl bg-white p-8 shadow-2xl animate-in fade-in zoom-in duration-200">
            <h3 className="mb-6 text-2xl font-bold text-gray-800">新增 Agency</h3>
            
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700">公司名稱</label>
                <input 
                  type="text" 
                  value={newAgency.name}
                  onChange={(e) => setNewAgency({...newAgency, name: e.target.value})}
                  className="mt-1 w-full rounded-md border border-gray-300 p-2 outline-none focus:ring-2 focus:ring-indigo-500"
                  placeholder="例如：太極高經紀公司"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">代幣符號</label>
                <input 
                  type="text" 
                  value={newAgency.token}
                  onChange={(e) => setNewAgency({...newAgency, token: e.target.value})}
                  className="mt-1 w-full rounded-md border border-gray-300 p-2 outline-none focus:ring-2 focus:ring-indigo-500"
                  placeholder="例如：TGO"
                />
              </div>
            </div>

            <div className="mt-8 flex justify-end space-x-3">
              <button 
                onClick={() => setIsModalOpen(false)}
                className="rounded-md px-4 py-2 text-gray-600 hover:bg-gray-100 transition-colors"
              >
                取消
              </button>
              <button 
                onClick={handleAddAgency}
                className="rounded-md bg-indigo-600 px-6 py-2 text-white hover:bg-indigo-700 shadow-md transition-colors"
              >
                確認新增
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default App;