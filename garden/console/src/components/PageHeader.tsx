interface Props {
  title: string;
  lede: string;
  right?: React.ReactNode;
}

export default function PageHeader({ title, lede, right }: Props) {
  return (
    <header className="page-head reveal">
      <div>
        <h1 className="page-title">{title}</h1>
        <p className="page-lede">{lede}</p>
      </div>
      {right && <div className="page-head-right">{right}</div>}
    </header>
  );
}
