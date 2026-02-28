import { render, screen } from "@testing-library/react";
import ChatMessages from "@/components/ChatMessages";
import type { MessageData } from "@/components/Message";

// Mock scrollIntoView
const scrollIntoViewMock = jest.fn();
window.HTMLElement.prototype.scrollIntoView = scrollIntoViewMock;

describe("ChatMessages", () => {
  beforeEach(() => {
    scrollIntoViewMock.mockClear();
  });

  it("shows welcome message when no messages", () => {
    render(<ChatMessages messages={[]} loading={false} />);
    expect(screen.getByText("Welcome to Wessley")).toBeInTheDocument();
  });

  it("renders messages", () => {
    const msgs: MessageData[] = [
      { role: "user", content: "Where is the fuse box?" },
      { role: "assistant", content: "The fuse box is under the dashboard." },
    ];
    render(<ChatMessages messages={msgs} loading={false} />);
    expect(screen.getByText("Where is the fuse box?")).toBeInTheDocument();
    expect(screen.getByText("The fuse box is under the dashboard.")).toBeInTheDocument();
  });

  it("shows loading indicator when loading", () => {
    render(<ChatMessages messages={[]} loading={true} />);
    expect(screen.getByText("Wessley")).toBeInTheDocument();
    // Loading dots should be present
    const dots = document.querySelectorAll(".typing-dot");
    expect(dots.length).toBe(3);
  });

  it("hides welcome when there are messages", () => {
    const msgs: MessageData[] = [{ role: "user", content: "hi" }];
    render(<ChatMessages messages={msgs} loading={false} />);
    expect(screen.queryByText("Welcome to Wessley")).not.toBeInTheDocument();
  });

  it("calls scrollIntoView when messages change", () => {
    const { rerender } = render(<ChatMessages messages={[]} loading={false} />);
    scrollIntoViewMock.mockClear();
    rerender(
      <ChatMessages
        messages={[{ role: "user", content: "hi" }]}
        loading={false}
      />
    );
    expect(scrollIntoViewMock).toHaveBeenCalled();
  });
});
