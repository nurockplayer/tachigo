import { useNavigate } from 'react-router'

const MOCK_STREAMERS = [
  {
    id: '1',
    name: 'nurockplayer',
    dailyHours: 3.5,
    activeMiners: 1240,
    totalPoints: 3200,
    multiplier: 3,
  },
  {
    id: '2',
    name: 'streamer_b',
    dailyHours: 2.0,
    activeMiners: 890,
    totalPoints: 2100,
    multiplier: 2,
  },
  {
    id: '3',
    name: 'streamer_c',
    dailyHours: 5.1,
    activeMiners: 2300,
    totalPoints: 6800,
    multiplier: 5,
  },
]

export default function StreamersPage() {
  const navigate = useNavigate()

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-foreground">實況主管理</h1>

      <div className="overflow-hidden rounded-lg border border-border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border bg-secondary/50">
              <th className="px-4 py-3 text-left font-medium text-muted-foreground">實況主</th>
              <th className="px-4 py-3 text-right font-medium text-muted-foreground">本日開台</th>
              <th className="px-4 py-3 text-right font-medium text-muted-foreground">挖礦觀眾</th>
              <th className="px-4 py-3 text-right font-medium text-muted-foreground">總產出點數</th>
              <th className="px-4 py-3 text-right font-medium text-muted-foreground">倍率</th>
              <th className="px-4 py-3" />
            </tr>
          </thead>
          <tbody>
            {MOCK_STREAMERS.map((streamer, index) => (
              <tr
                key={streamer.id}
                className={`cursor-pointer border-b border-border transition-colors last:border-0 hover:bg-accent/30 ${index % 2 === 0 ? '' : 'bg-secondary/20'}`}
                onClick={() => navigate(`/streamers/${streamer.id}`)}
              >
                <td className="px-4 py-3 font-medium text-foreground">{streamer.name}</td>
                <td className="px-4 py-3 text-right text-muted-foreground">
                  {streamer.dailyHours} hr
                </td>
                <td className="px-4 py-3 text-right text-muted-foreground">
                  {streamer.activeMiners.toLocaleString()}
                </td>
                <td className="px-4 py-3 text-right text-muted-foreground">
                  {streamer.totalPoints.toLocaleString()}
                </td>
                <td className="px-4 py-3 text-right text-muted-foreground">
                  {streamer.multiplier}x
                </td>
                <td className="px-4 py-3 text-right text-muted-foreground">→</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <p className="text-xs text-muted-foreground">* 目前顯示示範資料，後端 API 完成後自動串接</p>
    </div>
  )
}
