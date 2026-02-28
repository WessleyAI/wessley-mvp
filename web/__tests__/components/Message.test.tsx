import { render, screen } from "@testing-library/react";
import Message from "@/components/Message";
import type { MessageData } from "@/components/Message";

describe("Message", () => {
  it("renders user message with user styling (right-aligned)", () => {
    const msg: MessageData = { role: "user", content: "Hello" };
    const { container } = render(<Message msg={msg} />);
    const wrapper = container.firstChild as HTMLElement;
    expect(wrapper.className).toContain("justify-end");
    expect(screen.getByText("Hello")).toBeInTheDocument();
  });

  it("renders assistant message with assistant styling (left-aligned)", () => {
    const msg: MessageData = { role: "assistant", content: "Hi there" };
    const { container } = render(<Message msg={msg} />);
    const wrapper = container.firstChild as HTMLElement;
    expect(wrapper.className).toContain("justify-start");
    expect(screen.getByText("Hi there")).toBeInTheDocument();
  });

  it("shows 'Wessley' label for assistant messages", () => {
    const msg: MessageData = { role: "assistant", content: "Answer" };
    render(<Message msg={msg} />);
    expect(screen.getByText("Wessley")).toBeInTheDocument();
  });

  it("does not show 'Wessley' label for user messages", () => {
    const msg: MessageData = { role: "user", content: "Question" };
    render(<Message msg={msg} />);
    expect(screen.queryByText("Wessley")).not.toBeInTheDocument();
  });

  it("renders sources when provided", () => {
    const msg: MessageData = {
      role: "assistant",
      content: "Answer",
      sources: [
        { title: "Manual Page 42", source_type: "pdf", score: 0.95, url: "https://example.com" },
      ],
    };
    render(<Message msg={msg} />);
    expect(screen.getByText("Manual Page 42")).toBeInTheDocument();
    expect(screen.getByText("95%")).toBeInTheDocument();
  });

  it("preserves whitespace in content", () => {
    const msg: MessageData = { role: "assistant", content: "Line 1\nLine 2" };
    render(<Message msg={msg} />);
    expect(screen.getByText((_, el) => el?.textContent === "Line 1\nLine 2")).toBeInTheDocument();
  });
});
