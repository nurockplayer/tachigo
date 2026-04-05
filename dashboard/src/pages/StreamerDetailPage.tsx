import { useNavigate, useParams } from 'react-router'

const MOCK_DETAIL = {
  name: 'nurockplayer',
  currentSessionHours: 1.5,
  dailyHours: 3.5,
  monthlyHours: 28.0,
  yearlyHours: 120.0,
  uniqueMiners: 1240,
  avgSessionMinutes: 18,
  totalTokenMinted: 3200,
  secondsPerPoint: 60,
  multiplier: 3,
}

function TimeCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex-1 rounded-lg border border-border bg-secondary/30 p-4 text-center">
      <p className="mb-1 text-xs text-muted-foreground">{label}</p>
      <p className="text-lg font-bold text-foreground">{value}</p>
    </div>
  )
}

function MetricCard({
  label,
  value,
  unit,
}: {
  label: string
  value: string | number
  unit: string
}) {
  return (
    <div className="flex-1 space-y-1 rounded-lg border border-border bg-secondary/30 p-5">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-2xl font-bold text-foreground">
        {typeof value === 'number' ? value.toLocaleString() : value}
        <span className="ml-1 text-sm font-normal text-muted-foreground">{unit}</span>
      </p>
    </div>
  )
}

export default function StreamerDetailPage() {
  const { streamerId } = useParams()
  const navigate = useNavigate()
  const detail = MOCK_DETAIL
  const minuteOutput = ((60 / detail.secondsPerPoint) * detail.multiplier).toFixed(1)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <button
            onClick={() => navigate('/streamers')}
            className="text-sm text-muted-foreground transition-colors hover:text-foreground"
          >
            ← 返回列表
          </button>
          <h1 className="text-2xl font-bold text-foreground">{detail.name}</h1>
          <span className="text-xs text-muted-foreground">ID: {streamerId}</span>
        </div>
        <div className="flex gap-2">
          <button
            disabled
            className="cursor-not-allowed rounded-md border border-border px-4 py-2 text-sm text-muted-foreground opacity-50"
          >
            空投
          </button>
          <button
            disabled
            className="cursor-not-allowed rounded-md border border-border px-4 py-2 text-sm text-muted-foreground opacity-50"
          >
            調整倍率
          </button>
        </div>
      </div>

      <section className="space-y-3">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
          開台時數
        </h2>
        <div className="flex gap-3">
          <TimeCard label="本次" value={`${detail.currentSessionHours} hr`} />
          <TimeCard label="本日" value={`${detail.dailyHours} hr`} />
          <TimeCard label="本月" value={`${detail.monthlyHours} hr`} />
          <TimeCard label="年度" value={`${detail.yearlyHours} hr`} />
        </div>
      </section>

      <div className="flex gap-3">
        <MetricCard label="挖礦參與人數" value={detail.uniqueMiners} unit="人" />
        <MetricCard label="觀眾平均停留" value={detail.avgSessionMinutes} unit="分" />
        <MetricCard label="總產出點數" value={detail.totalTokenMinted} unit="點" />
      </div>

      <section className="space-y-4 rounded-lg border border-border bg-secondary/30 p-6">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
          挖礦倍率設定
        </h2>
        <div className="space-y-2 text-sm">
          <div className="flex justify-between">
            <span className="text-muted-foreground">每秒點數基準</span>
            <span className="font-medium text-foreground">
              {detail.secondsPerPoint} 秒 / 點
            </span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">目前倍率</span>
            <span className="font-medium text-foreground">{detail.multiplier}x</span>
          </div>
          <div className="mt-2 flex justify-between border-t border-border pt-2">
            <span className="text-muted-foreground">每分鐘產出</span>
            <span className="font-semibold text-primary">
              {minuteOutput} 點
              <span className="ml-1 text-xs text-muted-foreground">
                (60/{detail.secondsPerPoint} × {detail.multiplier})
              </span>
            </span>
          </div>
        </div>
      </section>

      <p className="text-xs text-muted-foreground">* 目前顯示示範資料，後端 API 完成後自動串接</p>
    </div>
  )
}
