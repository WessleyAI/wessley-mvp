export default function Header() {
  return (
    <header className="sticky top-0 z-10 border-b border-border bg-bg/80 backdrop-blur-sm">
      <div className="mx-auto flex h-14 max-w-3xl items-center px-4">
        <span className="text-xl">⚡</span>
        <h1 className="ml-2 text-lg font-semibold text-white">
          Wessley <span className="font-normal text-slate-400">— Vehicle AI</span>
        </h1>
      </div>
    </header>
  );
}
