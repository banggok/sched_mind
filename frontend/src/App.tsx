export function App() {
  return (
    <main className="grid min-h-screen place-items-center bg-stone-950 px-6 py-16 text-stone-100">
      <section
        className="w-full max-w-3xl rounded-3xl border border-white/10 bg-white/5 p-8 shadow-2xl shadow-emerald-950/40 backdrop-blur sm:p-12"
        aria-labelledby="page-title"
      >
        <p className="mb-4 text-sm font-semibold tracking-[0.2em] text-emerald-400 uppercase">
          Scheduling, clearly organized
        </p>
        <h1
          id="page-title"
          className="text-6xl font-black tracking-tight text-balance sm:text-8xl"
        >
          sched_mind
        </h1>
        <p className="mt-6 max-w-xl text-lg leading-8 text-stone-300">
          The React and Tailwind foundation is ready for scheduling
          requirements.
        </p>
      </section>
    </main>
  )
}

