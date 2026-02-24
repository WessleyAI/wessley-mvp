import { render, screen } from "@testing-library/react";
import Sources from "@/components/Sources";
import type { Source } from "@/lib/api";

describe("Sources", () => {
  it("renders nothing when sources array is empty", () => {
    const { container } = render(<Sources sources={[]} />);
    expect(container.innerHTML).toBe("");
  });

  it("renders nothing when sources is undefined", () => {
    // @ts-expect-error testing undefined
    const { container } = render(<Sources sources={undefined} />);
    expect(container.innerHTML).toBe("");
  });

  it("renders citations with titles", () => {
    const sources: Source[] = [
      { title: "Wiring Diagram", source_type: "pdf", score: 0.9 },
      { title: "Service Manual", source_type: "web", score: 0.85, url: "https://example.com" },
    ];
    render(<Sources sources={sources} />);
    expect(screen.getByText("Sources")).toBeInTheDocument();
    expect(screen.getByText("Wiring Diagram")).toBeInTheDocument();
    expect(screen.getByText("Service Manual")).toBeInTheDocument();
  });

  it("displays relevance scores as percentages", () => {
    const sources: Source[] = [
      { title: "Test", source_type: "pdf", score: 0.873 },
    ];
    render(<Sources sources={sources} />);
    expect(screen.getByText("87%")).toBeInTheDocument();
  });

  it("renders links for sources with URLs", () => {
    const sources: Source[] = [
      { title: "Online Source", source_type: "web", score: 0.8, url: "https://example.com/page" },
    ];
    render(<Sources sources={sources} />);
    const link = screen.getByText("Online Source");
    expect(link.tagName).toBe("A");
    expect(link).toHaveAttribute("href", "https://example.com/page");
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("renders span (not link) for sources without URLs", () => {
    const sources: Source[] = [
      { title: "Local Doc", source_type: "pdf", score: 0.7 },
    ];
    render(<Sources sources={sources} />);
    const el = screen.getByText("Local Doc");
    expect(el.tagName).toBe("SPAN");
  });

  it("falls back to source_type when title is missing", () => {
    const sources: Source[] = [
      { title: "", source_type: "pdf", score: 0.6 },
    ];
    render(<Sources sources={sources} />);
    expect(screen.getByText("pdf")).toBeInTheDocument();
  });
});
