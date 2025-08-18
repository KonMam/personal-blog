# mamonas.dev - Personal Blog

This is my personal blog, built with [Hugo](https://gohugo.io/) and the [re-Terminal](https://github.com/mirus-ua/hugo-theme-re-terminal) theme. The blog is automatically deployed to [GitHub Pages](https://pages.github.com/) using GitHub Actions.

## Technology Stack

*   **Static Site Generator:** [Hugo](https://gohugo.io/)
*   **Theme:** [re-Terminal](https://github.com/mirus-ua/hugo-theme-re-terminal)
*   **Hosting:** [GitHub Pages](https://pages.github.com/)
*   **CI/CD:** [GitHub Actions](https://github.com/features/actions)

## Usage Notes

### Local Development

To run the blog locally, you need to have Hugo installed. If you don't have it, you can find the installation instructions [here](https://gohugo.io/getting-started/installing/).

Once you have Hugo installed, navigate to the root directory of this project in your terminal and run the following command to start the local server:

```bash
hugo server
```

The blog will be available at `http://localhost:1313/`. Any changes you make to the content or templates will be automatically reloaded in your browser.

### Creating New Content

#### New Blog Post

To create a new blog post, use the following command:

```bash
hugo new content posts/your-new-post-title.md
```

This will create a new Markdown file in the `content/posts` directory. Open the file, update the front matter (title, date, description, tags, `draft: false`), and start writing your post.

#### New Standalone Page (e.g., About)

To create a new standalone page (like the "About" page), use a similar command:

```bash
hugo new content about.md
```

This will create `content/about.md`. You can then edit this file. To add it to the navigation menu, you'll need to edit `hugo.toml` under the `[languages.en.menu.main]` section.

### Deployment

This blog is set up for continuous deployment using GitHub Actions. Whenever you push or merge changes to the `main` branch of your GitHub repository, the workflow will automatically:

1.  Build your Hugo site.
2.  Deploy the generated static files to the `gh-pages` branch.
3.  Update your live site on GitHub Pages.

**To deploy your changes, simply commit them and push to the `main` branch:**

```bash
git add .
git commit -m "Your commit message"
git push origin main
```
